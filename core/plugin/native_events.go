package plugin

import (
	"sort"

	abi "GoCraft/abi/v1"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

const EventBlockBreak = "block.break"

// EmitBlockBreak publishes the same native payload for every client edition.
func (b *Bus) EmitBlockBreak(p *player.Player, pos spatial.BlockPos, block coreworld.Block, tool player.ItemStack) bool {
	if !b.hasSubscribers(EventBlockBreak) {
		return true
	}
	event := &abi.Event{
		Type:      EventBlockBreak,
		OnFailure: abi.FailureAllow,
		Fields: []abi.Value{
			playerReference(p),
			abi.List(abi.Int64(int64(pos.X)), abi.Int64(int64(pos.Y)), abi.Int64(int64(pos.Z))),
			blockValue(block),
			abi.String(tool.ItemID),
			b.injectedPermissions(EventBlockBreak, p),
		},
	}
	return b.EmitCancellable(event)
}

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
	sort.Strings(ordered)
	values := make([]abi.Value, 0, len(ordered))
	for _, node := range ordered {
		allowed := resolve != nil && p != nil && resolve(p, node)
		values = append(values, abi.List(abi.String(node), abi.Bool(allowed)))
	}
	return abi.List(values...)
}

func (b *Bus) hasSubscribers(event string) bool {
	b.mu.RLock()
	hasSubscribers := len(b.subs[event]) != 0
	b.mu.RUnlock()
	return hasSubscribers
}

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
