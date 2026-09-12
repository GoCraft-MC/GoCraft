package plugin

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type busMetrics struct {
	bus            *Bus
	disabled       *prometheus.Desc
	dispatch, cold *prometheus.HistogramVec
	failures       *prometheus.CounterVec
	starved        *prometheus.CounterVec
	respawns       *prometheus.CounterVec
}

// RegisterMetrics installs collectors once, before attaching plugins or
// dispatching events. Metric labels come from declared plugin subscriptions.
func (b *Bus) RegisterMetrics(registry prometheus.Registerer) {
	histogram := func(name, help string) *prometheus.HistogramVec {
		return prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "gocraft_plugin_" + name, Help: help,
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.1, 1},
		}, []string{"plugin", "event"})
	}
	counter := func(name, help string) *prometheus.CounterVec {
		return prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gocraft_plugin_" + name, Help: help,
		}, []string{"plugin", "event"})
	}
	m := &busMetrics{
		bus: b,
		disabled: prometheus.NewDesc("gocraft_plugin_disabled",
			"Whether a loaded plugin was disabled by its failure ratio.", []string{"plugin"}, nil),
		dispatch: histogram("event_dispatch_seconds", "Duration of plugin event dispatches in seconds, including failures."),
		cold:     histogram("cold_start_seconds", "Duration of first completed dispatches per plugin event subscription in seconds."),
		failures: counter("event_failures_total", "Total failed plugin event dispatches."),
		starved:  counter("event_starved_total", "Total plugin event dispatches skipped because the shared budget expired."),
		respawns: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gocraft_plugin_runtime_respawns_total", Help: "Total successful plugin runtime respawns.",
		}, []string{"runtime"}),
	}
	registry.MustRegister(m, m.dispatch, m.cold, m.failures, m.starved, m.respawns)
	b.metrics = m
}

func (m *busMetrics) Describe(ch chan<- *prometheus.Desc) { ch <- m.disabled }

func (m *busMetrics) Collect(ch chan<- prometheus.Metric) {
	m.bus.mu.RLock()
	trackers := make(map[string]*healthTracker, len(m.bus.health))
	for id, tracker := range m.bus.health {
		trackers[id] = tracker
	}
	m.bus.mu.RUnlock()
	for id, tracker := range trackers {
		disabled := 0.0
		if tracker.isDisabled() {
			disabled = 1
		}
		ch <- prometheus.MustNewConstMetric(m.disabled, prometheus.GaugeValue, disabled, id)
	}
}

func (b *Bus) recordDispatch(sub *subscriber, event string, took time.Duration, failed, cold bool) {
	sub.health.record(time.Now(), failed, took)
	if m := b.metrics; m != nil {
		m.dispatch.WithLabelValues(sub.id, event).Observe(took.Seconds())
		m.failures.WithLabelValues(sub.id, event).Add(boolCount(failed))
		if cold {
			sub.metricsCold.Do(func() { m.cold.WithLabelValues(sub.id, event).Observe(took.Seconds()) })
		}
	}
}

func boolCount(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

// RecordRuntimeRespawn records a completed host-managed runtime restart.
func (b *Bus) RecordRuntimeRespawn(runtime string) {
	if b.metrics != nil {
		b.metrics.respawns.WithLabelValues(runtime).Inc()
	}
}
