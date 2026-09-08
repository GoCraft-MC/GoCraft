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
	took   time.Duration
}

type healthTracker struct {
	mu       sync.Mutex
	samples  []healthSample
	failures int
	total    time.Duration
	starved  map[string]uint64
	disabled bool
}

// HealthSnapshot is a point-in-time view of one plugin's event health.
type HealthSnapshot struct {
	Calls           int
	Failures        int
	AverageDuration time.Duration
	Starved         map[string]uint64
	Disabled        bool
}

func newHealthTracker() *healthTracker {
	return &healthTracker{starved: make(map[string]uint64)}
}

func (h *healthTracker) record(now time.Time, failed bool, took time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prune(now)
	h.samples = append(h.samples, healthSample{at: now, failed: failed, took: took})
	h.total += took
	if failed {
		h.failures++
	}
	if len(h.samples) >= minimumHealthSamples && float64(h.failures)/float64(len(h.samples)) > maximumFailureRatio {
		h.disabled = true
	}
}

func (h *healthTracker) recordStarved(event string) {
	h.mu.Lock()
	h.starved[event]++
	h.mu.Unlock()
}

func (h *healthTracker) isDisabled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.disabled
}

func (h *healthTracker) snapshot(now time.Time) HealthSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prune(now)
	snapshot := HealthSnapshot{Calls: len(h.samples), Failures: h.failures, Starved: make(map[string]uint64), Disabled: h.disabled}
	if snapshot.Calls != 0 {
		snapshot.AverageDuration = h.total / time.Duration(snapshot.Calls)
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
		h.total -= h.samples[first].took
		if h.samples[first].failed {
			h.failures--
		}
		first++
	}
	h.samples = h.samples[first:]
}
