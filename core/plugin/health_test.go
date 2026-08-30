package plugin

import (
	"testing"
	"time"
)

func TestHealthTrackerDisablesHighFailureRatio(t *testing.T) {
	tracker := newHealthTracker()
	now := time.Unix(100, 0)
	for index := range minimumHealthSamples {
		tracker.record(now.Add(time.Duration(index)*time.Millisecond), index < 2, time.Millisecond)
	}
	snapshot := tracker.snapshot(now.Add(time.Second))
	if snapshot.Calls != minimumHealthSamples || snapshot.Failures != 2 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if !snapshot.Disabled {
		t.Fatal("plugin over the failure threshold stayed enabled")
	}
	if snapshot.AverageDuration != time.Millisecond {
		t.Fatalf("average duration = %s", snapshot.AverageDuration)
	}
}

func TestHealthTrackerUsesSlidingWindow(t *testing.T) {
	tracker := newHealthTracker()
	now := time.Unix(100, 0)
	tracker.record(now, true, time.Millisecond)
	tracker.record(now.Add(healthWindow+time.Second), false, 2*time.Millisecond)
	snapshot := tracker.snapshot(now.Add(healthWindow + time.Second))
	if snapshot.Calls != 1 || snapshot.Failures != 0 {
		t.Fatalf("expired samples remain: %+v", snapshot)
	}
}
