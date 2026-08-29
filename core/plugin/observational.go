package plugin

import (
	"log/slog"
	"time"

	abi "GoCraft/abi/v1"
)

// EmitObservational schedules an event without blocking the simulation tick.
func (b *Bus) EmitObservational(event *abi.Event) {
	if event == nil {
		return
	}
	subscribers := b.subscribers(event.Type)
	if len(subscribers) == 0 {
		return
	}
	event = cloneEvent(event)
	go b.dispatchObservational(event, subscribers)
}

func (b *Bus) dispatchObservational(event *abi.Event, subscribers []*subscriber) {
	for _, sub := range subscribers {
		if b.ctx.Err() != nil {
			return
		}
		if sub.health.isDisabled() {
			continue
		}
		started := time.Now()
		verdict, err := sub.instance.Dispatch(b.ctx, cloneEvent(event))
		took := time.Since(started)
		if err != nil {
			sub.health.record(time.Now(), true, took)
			slog.Warn("plugin observational event failed", "plugin", sub.id, "event", event.Type, "err", err)
			continue
		}
		sub.health.record(time.Now(), false, took)
		b.enqueueEffects(sub, event.Type, verdict.Effects)
	}
}

func cloneEvent(event *abi.Event) *abi.Event {
	cloned := *event
	cloned.Fields = make([]abi.Value, len(event.Fields))
	for index, field := range event.Fields {
		cloned.Fields[index] = cloneValue(field)
	}
	return &cloned
}
