package plugin

import (
	"context"
	"sync"
	"testing"
	"time"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
	"github.com/prometheus/client_golang/prometheus"
)

func TestConcurrentFirstEventsOnlyRecordOneColdStart(t *testing.T) {
	bus := NewBus(context.Background(), time.Second)
	registry := prometheus.NewPedanticRegistry()
	bus.RegisterMetrics(registry)
	entered, release := make(chan struct{}, 2), make(chan struct{})
	instance := &fakeInstance{
		manifest: gcpkg.Manifest{ID: "concurrent", Subscriptions: []gcpkg.Subscription{{Event: "player.join"}}},
		dispatch: func(context.Context, *abi.Event) (abi.Verdict, error) {
			entered <- struct{}{}
			<-release
			return abi.Verdict{}, nil
		},
	}
	if err := bus.Attach(instance); err != nil {
		t.Fatal(err)
	}
	var dispatches sync.WaitGroup
	var unblock sync.Once
	t.Cleanup(func() { unblock.Do(func() { close(release) }); dispatches.Wait() })
	for range 2 {
		dispatches.Go(func() {
			bus.dispatchObservational(&abi.Event{Type: "player.join"}, bus.subscribers("player.join"))
		})
	}
	for range 2 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("observational dispatches did not overlap")
		}
	}
	if _, err := registry.Gather(); err != nil {
		t.Fatal(err)
	}
	unblock.Do(func() { close(release) })
	dispatches.Wait()
	if got := histogramCount(t, registry, "gocraft_plugin_cold_start_seconds"); got != 1 {
		t.Fatalf("cold start samples = %d, want one per subscription", got)
	}
	if got := histogramCount(t, registry, "gocraft_plugin_event_dispatch_seconds"); got != 2 {
		t.Fatalf("dispatch samples = %d, want both events", got)
	}
}
