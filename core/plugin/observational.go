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

// EmitObservationalTo delivers an event to named plugins only.
//
// It exists for replay. When a runtime dies and comes back, its plugins have
// missed everything that happened while they were down and the host makes it up
// to them — §13's synthetic player.join for everyone already online. The
// plugins that never went away must not receive those: they saw the real joins,
// and a Lua plugin watching every player join again each time the JVM crashes
// would be counting arrivals that never happened.
//
// Only observational events can be replayed. A cancellable one is a question
// the host already answered, and asking it again after the fact would invite an
// answer nothing can act on.
func (b *Bus) EmitObservationalTo(event *abi.Event, pluginIDs []string) {
	if event == nil || len(pluginIDs) == 0 {
		return
	}
	wanted := make(map[string]struct{}, len(pluginIDs))
	for _, id := range pluginIDs {
		wanted[id] = struct{}{}
	}
	var targeted []*subscriber
	for _, sub := range b.subscribers(event.Type) {
		if _, ok := wanted[sub.id]; ok {
			targeted = append(targeted, sub)
		}
	}
	if len(targeted) == 0 {
		return
	}
	event = cloneEvent(event)
	go b.dispatchObservational(event, targeted)
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
