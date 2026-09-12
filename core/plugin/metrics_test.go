package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func histogramCount(t *testing.T, registry *prometheus.Registry, name string) uint64 {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == name {
			return family.Metric[0].GetHistogram().GetSampleCount()
		}
	}
	t.Fatalf("missing histogram %s", name)
	return 0
}

func TestPluginMetricsCoverEveryDispatchPath(t *testing.T) {
	for name, dispatch := range map[string]func(*Bus, *abi.Event){
		"cancellable": func(b *Bus, e *abi.Event) { b.EmitCancellable(e) },
		"custom": func(b *Bus, e *abi.Event) {
			b.EmitCustom(gcpkg.EventDefinition{Type: e.Type}, abi.Emission{PluginID: "emitter"})
		},
		"observational": func(b *Bus, e *abi.Event) { b.dispatchObservational(e, b.subscribers(e.Type)) },
	} {
		t.Run(name, func(t *testing.T) {
			registry := prometheus.NewPedanticRegistry()
			bus := NewBus(context.Background(), time.Second)
			bus.RegisterMetrics(registry)
			instance := &fakeInstance{
				manifest: gcpkg.Manifest{ID: "broken", Subscriptions: []gcpkg.Subscription{{Event: "test.event"}}},
				dispatch: func(context.Context, *abi.Event) (abi.Verdict, error) {
					return abi.Verdict{}, errors.New("runtime unavailable")
				},
			}
			if err := bus.Attach(instance); err != nil {
				t.Fatal(err)
			}
			if disabled := testutil.ToFloat64(bus.metrics); disabled != 0 {
				t.Fatalf("new plugin disabled = %v", disabled)
			}
			for range minimumHealthSamples + 1 {
				dispatch(bus, &abi.Event{Type: "test.event"})
			}
			bus.health["broken"].snapshot(time.Now().Add(2 * healthWindow))
			if got := testutil.ToFloat64(bus.metrics.failures.WithLabelValues("broken", "test.event")); got != minimumHealthSamples {
				t.Fatalf("lifetime failures after health window expired = %v", got)
			}
			if got := histogramCount(t, registry, "gocraft_plugin_event_dispatch_seconds"); got != minimumHealthSamples {
				t.Fatalf("dispatch samples = %d", got)
			}
			if got := histogramCount(t, registry, "gocraft_plugin_cold_start_seconds"); got != 1 {
				t.Fatalf("cold start samples = %d", got)
			}
			if disabled := testutil.ToFloat64(bus.metrics); disabled != 1 {
				t.Fatalf("failing plugin disabled = %v", disabled)
			}
			bus.RecordRuntimeRespawn("jvm")
			if got := testutil.ToFloat64(bus.metrics.respawns.WithLabelValues("jvm")); got != 1 {
				t.Fatalf("runtime respawns = %v", got)
			}
			bus.Detach("broken")
			if count := testutil.CollectAndCount(bus.metrics); count != 0 {
				t.Fatal("unloaded plugin still exports a disabled gauge")
			}
		})
	}
}
