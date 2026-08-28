package plugin

import (
	"sync"
	"time"
)

const (
	healthWindow         = time.Minute
	minimumHealthSamples = 10
	maximumFailureRatio  = 0.10
)

type healthSample struct {
	at     time.Time
	failed bool
}

type healthTracker struct {
	mu       sync.Mutex
	samples  []healthSample
	starved  map[string]uint64
	disabled bool
}

// HealthSnapshot is a point-in-time view of one plugin's event health.
type HealthSnapshot struct {
	Calls    int
	Failures int
	Starved  map[string]uint64
	Disabled bool
}

func newHealthTracker() *healthTracker {
	return &healthTracker{starved: make(map[string]uint64)}
}

func (h *healthTracker) record(now time.Time, failed bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prune(now)
	h.samples = append(h.samples, healthSample{at: now, failed: failed})
	failures := 0
	for _, sample := range h.samples {
		if sample.failed {
			failures++
		}
	}
	if len(h.samples) >= minimumHealthSamples && float64(failures)/float64(len(h.samples)) > maximumFailureRatio {
		h.disabled = true
	}
}

func (h *healthTracker) recordStarved(event string) {
	h.mu.Lock()
	h.starved[event]++
	h.mu.Unlock()
}

func (h *healthTracker) snapshot(now time.Time) HealthSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prune(now)
	snapshot := HealthSnapshot{Calls: len(h.samples), Starved: make(map[string]uint64), Disabled: h.disabled}
	for _, sample := range h.samples {
		if sample.failed {
			snapshot.Failures++
		}
	}
	for event, count := range h.starved {
		snapshot.Starved[event] = count
	}
	return snapshot
}

func (h *healthTracker) prune(now time.Time) {
	cutoff := now.Add(-healthWindow)
	first := 0
	for first < len(h.samples) && h.samples[first].at.Before(cutoff) {
		first++
	}
	h.samples = h.samples[first:]
}
