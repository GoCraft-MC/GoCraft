package gocraft

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

type TaskID uint64

// After runs a task once on its own goroutine after the delay.
func (s *Scheduler) After(delay time.Duration, task func()) (TaskID, error) {
	return s.schedule(delay, false, task)
}

// Every runs a task repeatedly on its own goroutine.
func (s *Scheduler) Every(interval time.Duration, task func()) (TaskID, error) {
	return s.schedule(interval, true, task)
}

func (s *Scheduler) schedule(delay time.Duration, repeat bool, task func()) (TaskID, error) {
	if delay <= 0 || task == nil {
		return 0, fmt.Errorf("gocraft: task and positive duration are required")
	}
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return 0, fmt.Errorf("gocraft: scheduler is disabled")
	}
	s.next++
	id := s.next
	ctx, cancel := context.WithCancel(context.Background())
	s.cancels[id] = cancel
	s.wait.Add(1)
	s.mu.Unlock()
	go s.run(ctx, id, delay, repeat, task)
	return id, nil
}

func (s *Scheduler) run(ctx context.Context, id TaskID, delay time.Duration, repeat bool, task func()) {
	defer s.wait.Done()
	defer s.remove(id)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.call(id, task)
			if !repeat {
				return
			}
			timer.Reset(delay)
		}
	}
}

func (s *Scheduler) remove(id TaskID) {
	s.mu.Lock()
	delete(s.cancels, id)
	s.mu.Unlock()
}

func (s *Scheduler) call(id TaskID, task func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.Error("plugin task panicked", "task", id,
				"panic", recovered, "stack", string(debug.Stack()))
		}
	}()
	task()
}

// Cancel stops an owned task. It reports whether the task was still active.
func (s *Scheduler) Cancel(id TaskID) bool {
	s.mu.Lock()
	cancel, ok := s.cancels[id]
	s.mu.Unlock()
	if ok {
		cancel()
	}
	return ok
}

func (s *Scheduler) stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	cancels := make([]context.CancelFunc, 0, len(s.cancels))
	for _, cancel := range s.cancels {
		cancels = append(cancels, cancel)
	}
	s.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	s.wait.Wait()
}
