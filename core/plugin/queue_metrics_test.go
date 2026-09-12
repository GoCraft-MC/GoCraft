package plugin

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMutationQueueMetricsTrackDepthAndRejectedCalls(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	queue := NewMutationQueue()
	queue.RegisterMetrics(registry)
	assertDepth := func(want int) {
		t.Helper()
		expected := fmt.Sprintf("# HELP gocraft_plugin_effect_queue_depth Number of plugin host calls waiting for the simulation tick.\n# TYPE gocraft_plugin_effect_queue_depth gauge\ngocraft_plugin_effect_queue_depth %d\n", want)
		if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "gocraft_plugin_effect_queue_depth"); err != nil {
			t.Fatal(err)
		}
	}
	assertDepth(0)
	if err := queue.Enqueue(abi.HostCall{}); err == nil {
		t.Fatal("invalid effect was accepted")
	}
	for range maximumQueuedCalls {
		if err := queue.Enqueue(abi.HostCall{Type: "message"}); err != nil {
			t.Fatal(err)
		}
	}
	assertDepth(maximumQueuedCalls)
	if err := queue.Enqueue(abi.HostCall{Type: "message"}); !errors.Is(err, ErrMutationQueueFull) {
		t.Fatalf("enqueue on full queue = %v", err)
	}
	count, err := queue.Drain(func(abi.HostCall) error { return nil })
	if err != nil || count != maximumQueuedCalls {
		t.Fatalf("drain = %d, %v", count, err)
	}
	assertDepth(0)
	queue.Close()
	if err := queue.Enqueue(abi.HostCall{Type: "message"}); !errors.Is(err, ErrMutationQueueClosed) {
		t.Fatalf("enqueue on closed queue = %v", err)
	}
	for _, reason := range []string{"invalid", "full", "closed"} {
		if got := testutil.ToFloat64(queue.metrics.rejected.WithLabelValues(reason)); got != 1 {
			t.Errorf("rejected %s = %v, want 1", reason, got)
		}
	}
}
