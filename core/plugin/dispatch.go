package plugin

import (
	"context"
	"errors"
	"log/slog"

	abi "GoCraft/abi/v1"
)

// EmitCancellable blocks for subscriber verdicts under one shared event budget.
func (b *Bus) EmitCancellable(event *abi.Event) bool {
	if event == nil {
		return true
	}
	subscribers := b.subscribers(event.Type)
	if len(subscribers) == 0 {
		return true
	}
	ctx, cancel := context.WithTimeout(b.ctx, b.budget)
	defer cancel()
	for _, sub := range subscribers {
		if ctx.Err() != nil {
			return failureAllows(event)
		}
		verdict, err := sub.instance.Dispatch(ctx, event)
		if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
			slog.Warn("plugin event deadline exceeded", "plugin", sub.id, "event", event.Type)
			return failureAllows(event)
		}
		if err != nil {
			slog.Warn("plugin event dispatch failed", "plugin", sub.id, "event", event.Type, "err", err)
			if !failureAllows(event) {
				return false
			}
			continue
		}
		if verdict.Cancelled {
			return false
		}
	}
	return true
}

func (b *Bus) subscribers(event string) []*subscriber {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]*subscriber(nil), b.subs[event]...)
}

func failureAllows(event *abi.Event) bool {
	return event.OnFailure != abi.FailureDeny
}
