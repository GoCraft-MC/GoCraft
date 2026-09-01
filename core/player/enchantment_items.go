package player

import "GoCraft/core/itemregistry"

// Supports reports whether the canonical vanilla item tag accepts itemID for
// this enchantment. Enchantment behavior and compatibility rules remain code.
func (e Enchantment) Supports(itemID string) bool {
	return itemregistry.HasTag(itemID, "minecraft:enchantable/"+e.SupportedItems)
}
