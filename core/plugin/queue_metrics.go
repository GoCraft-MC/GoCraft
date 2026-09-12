package plugin

import "github.com/prometheus/client_golang/prometheus"

type queueMetrics struct {
	rejected *prometheus.CounterVec
}

// RegisterMetrics installs queue collectors once, before any host calls arrive.
func (q *MutationQueue) RegisterMetrics(registry prometheus.Registerer) {
	rejected := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gocraft_plugin_effect_rejected_total",
		Help: "Total plugin host calls rejected before entering the mutation queue.",
	}, []string{"reason"})
	for _, reason := range []string{"invalid", "closed", "full"} {
		rejected.WithLabelValues(reason)
	}
	registry.MustRegister(rejected, prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "gocraft_plugin_effect_queue_depth",
		Help: "Number of plugin host calls waiting for the simulation tick.",
	}, func() float64 {
		q.mu.Lock()
		defer q.mu.Unlock()
		return float64(len(q.calls))
	}))
	q.metrics = &queueMetrics{rejected: rejected}
}

func (q *MutationQueue) recordRejected(reason string) {
	if q.metrics != nil {
		q.metrics.rejected.WithLabelValues(reason).Inc()
	}
}
