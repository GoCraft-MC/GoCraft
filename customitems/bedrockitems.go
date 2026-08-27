package customitems

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// BedrockItemEntries returns the protocol.ItemEntry slice that must be appended
// to the vanilla item registry before passing it to conn.StartGame().
//
// Each entry has ComponentBased=true and carries the item components that
// Bedrock clients use to render and handle the item (display name, icon,
// max stack size, hand-equipped flag).
func (m *Manager) BedrockItemEntries() []protocol.ItemEntry {
	entries := make([]protocol.ItemEntry, 0, len(m.items))
	for _, item := range m.items {
		maxStack := item.Def.MaxStackSize
		if maxStack <= 0 {
			maxStack = 64
		}

		components := map[string]any{
			"item_properties": map[string]any{
				"allow_off_hand":          true,
				"can_destroy_in_creative": false,
				"creative_category":       int32(4), // 4 = Items tab
				"creative_group":          "",
				"foil":                    false,
				"hand_equipped":           item.Def.HandEquipped,
				"max_stack_size":          int32(maxStack),
			},
			"minecraft:icon": map[string]any{
				"textures": map[string]any{
					"default": item.Namespace + "_" + item.ID,
				},
			},
			"minecraft:display_name": map[string]any{
				"value": stripTags(item.Def.DisplayName),
			},
		}

		entries = append(entries, protocol.ItemEntry{
			Name:           item.Key(),
			RuntimeID:      item.BedrockRuntimeID,
			ComponentBased: true,
			Version:        protocol.ItemEntryVersionDataDriven,
			Data:           components,
		})
	}
	return entries
}
