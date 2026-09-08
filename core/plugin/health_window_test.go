package plugin

import (
	"testing"
	"time"
)

func TestHealthRollingCountersMatchWindowAfterExpiry(t *testing.T) {
	h := newHealthTracker()
	start := time.Now()
	for i := range 1000 {
		h.record(start.Add(time.Duration(i)*time.Millisecond), i%100 == 0, time.Duration(i)*time.Microsecond)
	}
	for _, now := range []time.Time{start.Add(time.Second), start.Add(healthWindow + 500*time.Millisecond), start.Add(2 * healthWindow)} {
		snapshot := h.snapshot(now)
		failures := 0
		var total time.Duration
		for _, sample := range h.samples {
			if sample.failed {
				failures++
			}
			total += sample.took
		}
		if snapshot.Failures != failures || h.total != total {
			t.Fatalf("counters drifted: %+v total=%v", snapshot, h.total)
		}
		if snapshot.Calls > 0 && snapshot.AverageDuration != total/time.Duration(snapshot.Calls) {
			t.Fatal("average drifted")
		}
	}
	if h.failures != 0 || h.total != 0 {
		t.Fatal("expired counters retained")
	}
}
