package ipc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fastLiveness() Liveness {
	return Liveness{Every: 100 * time.Millisecond, Timeout: 100 * time.Millisecond, Missed: 3}
}

// The failure this whole file exists for: a process that is alive, holding its
// socket open, and no longer answering. Nothing else in the package can see it.
func TestWatchKillsARuntimeThatStopsAnswering(t *testing.T) {
	child, err := Start(t.Context(), fakeConfig(t, "deaf"))
	if err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	err = child.Watch(t.Context(), fastLiveness())
	elapsed := time.Since(started)

	if !errors.Is(err, ErrUnresponsive) {
		t.Fatalf("Watch() = %v, want ErrUnresponsive", err)
	}
	// Three pings at 100ms with a 100ms timeout each: under a second, and not
	// instant either — one slow reply must not condemn a runtime.
	if elapsed < 200*time.Millisecond {
		t.Fatalf("Watch() gave up after %s, before the misses could accumulate", elapsed)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Watch() took %s to notice", elapsed)
	}
	select {
	case <-child.Exited():
	case <-time.After(10 * time.Second):
		t.Fatal("the unresponsive runtime was not killed")
	}
}

// A healthy runtime must survive being watched, however long the watch lasts.
func TestWatchLeavesAHealthyRuntimeAlone(t *testing.T) {
	child, err := Start(t.Context(), fakeConfig(t, "ok"))
	if err != nil {
		t.Fatal(err)
	}
	defer child.Stop(t.Context())

	ctx, cancel := context.WithTimeout(t.Context(), 700*time.Millisecond)
	defer cancel()
	if err := child.Watch(ctx, fastLiveness()); err != nil {
		t.Fatalf("Watch() = %v, want nil when the caller stops watching", err)
	}
	select {
	case <-child.Exited():
		t.Fatal("a healthy runtime was killed")
	default:
	}
}

// Watching stops when the process leaves on its own, without waiting for the
// heartbeat to notice.
func TestWatchReturnsWhenTheRuntimeExits(t *testing.T) {
	child, err := Start(t.Context(), fakeConfig(t, "quit"))
	if err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	err = child.Watch(t.Context(), fastLiveness())
	if err == nil {
		t.Fatal("Watch() returned nil for a runtime that exited")
	}
	if errors.Is(err, ErrUnresponsive) {
		t.Fatalf("Watch() = %v, want the exit rather than a heartbeat verdict", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("Watch() took %s to notice the exit", elapsed)
	}
}

// Answering a ping with something other than a pong is a protocol fault, not a
// slow reply: it will not fix itself, so it is not counted against the budget.
func TestWatchRejectsAWrongAnswerToAPing(t *testing.T) {
	child, err := Start(t.Context(), fakeConfig(t, "rude"))
	if err != nil {
		t.Fatal(err)
	}
	defer child.Stop(t.Context())

	err = child.Watch(t.Context(), fastLiveness())
	if !errors.Is(err, ErrProtocol) {
		t.Fatalf("Watch() = %v, want ErrProtocol", err)
	}
}

func TestLivenessDefaultsMatchTheDesign(t *testing.T) {
	every, timeout, missed := Liveness{}.resolve()
	if every != time.Second {
		t.Fatalf("default interval = %s, want one ping per second", every)
	}
	if timeout != every {
		t.Fatalf("default timeout = %s, want the interval", timeout)
	}
	if missed != 3 {
		t.Fatalf("default missed = %d, want three", missed)
	}
}
