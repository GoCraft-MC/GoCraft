package plugin

import (
	"log/slog"
	"math"

	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func (b *Bus) applyNativeMutations(sub *subscriber, event *abi.Event, mutations []abi.Mutation) {
	for _, mutation := range mutations {
		if !nativeMutationAllowed(event.Type, mutation) ||
			(mutation.Value.Kind == abi.ValueDouble && (math.IsNaN(mutation.Value.Double) || math.IsInf(mutation.Value.Double, 0))) {
			slog.Warn("plugin native mutation refused", "plugin", sub.id, "event", event.Type, "path", mutation.Path)
			continue
		}
		updated, err := abi.ApplyPath(event.Fields, mutation)
		if err != nil {
			slog.Warn("plugin native mutation refused", "plugin", sub.id, "event", event.Type, "err", err)
			continue
		}
		event.Fields = updated
	}
}
