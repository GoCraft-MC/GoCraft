package gocraft

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerOwnsAndCancelsTasks(t *testing.T) {
	scheduler := newScheduler(testLogger(&bytes.Buffer{}))
	var calls atomic.Int32
	ran := make(chan struct{}, 1)
	id, err := scheduler.Every(time.Millisecond, func() {
		calls.Add(1)
		select {
		case ran <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	<-ran
	if !scheduler.Cancel(id) {
		t.Fatal("active task was not cancelled")
	}
	scheduler.stop()
	got := calls.Load()
	time.Sleep(5 * time.Millisecond)
	if calls.Load() != got {
		t.Fatal("task ran after scheduler shutdown")
	}
	if _, err := scheduler.After(time.Millisecond, func() {}); err == nil {
		t.Fatal("disabled scheduler accepted a task")
	}
}

func TestSchedulerRecoversPanics(t *testing.T) {
	var output bytes.Buffer
	scheduler := newScheduler(testLogger(&output))
	ran := make(chan struct{})
	if _, err := scheduler.After(time.Millisecond, func() {
		close(ran)
		panic("task exploded")
	}); err != nil {
		t.Fatal(err)
	}
	<-ran
	scheduler.stop()
	if log := output.String(); !strings.Contains(log, "task exploded") || !strings.Contains(log, "stack") {
		t.Fatalf("panic log = %q", log)
	}
}
