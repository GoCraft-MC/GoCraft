package player

// InventorySize is the total number of slots in a Java Edition player inventory.
//
// Slot layout:
//
//	0      crafting output
//	1–4    2×2 crafting grid
//	5–8    armour (helmet, chestplate, leggings, boots)
//	9–35   main inventory (3 rows × 9)
//	36–44  hotbar (9 slots; index = HotbarStart + heldSlot)
//	45     off-hand
const InventorySize = 46

// HotbarStart is the slot index of the first hotbar slot.
const HotbarStart = 36

// ItemStack is a quantity of one item type occupying a single inventory slot.
// A zero-value ItemStack (or one with Count ≤ 0) represents an empty slot.
//
// ItemID uses the Minecraft resource-location format ("namespace:name"),
// e.g. "minecraft:stone".  Edition-specific numeric item IDs are resolved at
// the Java adapter boundary (java/world/items.go) and are not stored here.
type ItemStack struct {
	// ItemID is the canonical resource location of the item, e.g. "minecraft:stone".
	// Empty string means the slot is empty.
	ItemID string
	// Count is the number of items in the stack.
	Count int
}

// IsEmpty reports whether the slot contains no item.
func (s ItemStack) IsEmpty() bool {
	return s.Count <= 0 || s.ItemID == ""
}
