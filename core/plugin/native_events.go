package plugin

import (
	"sort"

	abi "GoCraft/abi/v1"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

// The hand-written half of the event system.
//
// The emitters and their type constants are generated into events.gen.go from
// abi/v1/events.proto, so the payload the host writes and the accessors every
// runtime reads come out of one source. What stays here is the vocabulary —
// the dozen or so stable types §03 keeps by hand because ergonomics matter and
// generated code would be mediocre — plus the two bus helpers the generated
// emitters call.
//
// A converter here decides the shape of a vocabulary type on the wire. Changing
// one changes what every runtime reads, so it is a wire break even though no
// schema moved.

// hasSubscribers is what makes an unsubscribed event free. The bus would reach
// the same answer, but only after building a payload nobody is waiting for.
func (b *Bus) hasSubscribers(event string) bool {
	b.mu.RLock()
	hasSubscribers := len(b.subs[event]) != 0
	b.mu.RUnlock()
	return hasSubscribers
}

// injectedPermissions resolves every node subscribed to this event, before the
// event is dispatched.
//
// This is what lets a handler answer can() from a map rather than by asking the
// server — a round trip it cannot afford, holding the tick while it waits. Only
// nodes some manifest declared are resolved, so a plugin querying one it never
// declared reads false.
func (b *Bus) injectedPermissions(event string, p *player.Player) abi.Value {
	b.mu.RLock()
	nodes := make(map[string]struct{})
	for _, sub := range b.subs[event] {
		for _, node := range sub.permissions {
			nodes[node] = struct{}{}
		}
	}
	resolve := b.permissionResolver
	b.mu.RUnlock()

	ordered := make([]string, 0, len(nodes))
	for node := range nodes {
		ordered = append(ordered, node)
	}
	// Sorted so the payload is byte-identical for the same inputs, whatever
	// order the map happened to iterate in.
	sort.Strings(ordered)
	values := make([]abi.Value, 0, len(ordered))
	for _, node := range ordered {
		allowed := resolve != nil && p != nil && resolve(p, node)
		values = append(values, abi.List(abi.String(node), abi.Bool(allowed)))
	}
	return abi.List(values...)
}

// playerReference is the PlayerRef vocabulary type: uuid, username, edition.
//
// An absent player is an empty list rather than a null, because the wire format
// has no null and a runtime reading a fixed layout would have to special-case
// one anyway.
func playerReference(p *player.Player) abi.Value {
	if p == nil {
		return abi.List()
	}
	edition := "java"
	if p.Edition == player.ClientEditionBedrock {
		edition = "bedrock"
	}
	return abi.List(abi.Bytes(p.UUID[:]), abi.String(p.Username), abi.String(edition))
}

// positionValue is the BlockPos vocabulary type.
func positionValue(pos spatial.BlockPos) abi.Value {
	return abi.List(abi.Int64(int64(pos.X)), abi.Int64(int64(pos.Y)), abi.Int64(int64(pos.Z)))
}

// blockValue is the Block vocabulary type: the full state, not a handle.
//
// Sending the whole thing means a handler never re-resolves what it was just
// given, which would be a round trip inside an event that is blocking the tick.
func blockValue(block coreworld.Block) abi.Value {
	keys := make([]string, 0, len(block.Properties))
	for key := range block.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	properties := make([]abi.Value, 0, len(keys))
	for _, key := range keys {
		properties = append(properties, abi.List(abi.String(key), abi.String(block.Properties[key])))
	}
	return abi.List(abi.String(block.ResourceLocation()), abi.List(properties...))
}
