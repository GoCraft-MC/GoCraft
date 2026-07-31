package world

import coreworld "GoCraft/core/world"

// itemIDs maps canonical item resource locations to their numeric protocol IDs
// for Minecraft Java Edition 1.21.4 (protocol 769).
//
// Populated at init time by registry.go from the embedded items.json.
// Replacing the JSON file with the full Minecraft data-generator output updates
// all IDs without changing Go source.
var itemIDs map[string]int32

// itemNames is the reverse of itemIDs: numeric item protocol ID → resource location.
// Populated at init time by registry.go alongside itemIDs.
var itemNames map[int32]string

// ItemName returns the resource location for a numeric item protocol ID, or ""
// if the item is not in the registry.
func ItemName(id int32) string {
	return itemNames[id]
}

// ItemID returns the numeric protocol ID for an item resource location, or -1
// if the item is not in the registry.
func ItemID(name string) int32 {
	id, ok := itemIDs[name]
	if !ok {
		return -1
	}
	return id
}

// IsPlaceableAsBlock reports whether an item can be placed as a block.
// The heuristic: if the item's resource location has a non-zero block state ID,
// it corresponds to a block.  Tools, food, etc. map to state ID 0 (air) and
// are therefore not placeable.
func IsPlaceableAsBlock(itemID string) bool {
	if itemID == "" || itemID == "minecraft:air" {
		return false
	}
	// Reuse the block state ID table; state ID 0 = air = not a block item.
	return StateID(ItemIDToBlock(itemID)) != 0
}

// ItemIDToBlock returns the canonical Block for an item resource location.
// For most block items the block shares the same resource location as the item.
func ItemIDToBlock(itemID string) coreworld.Block {
	ns, name := "minecraft", itemID
	if i := len("minecraft:"); len(itemID) > i && itemID[:i] == "minecraft:" {
		name = itemID[i:]
	}
	return coreworld.Block{Namespace: ns, Name: name}
}
