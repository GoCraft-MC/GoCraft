package world

import coreworld "GoCraft/core/world"

// itemIDs maps canonical item resource locations to their numeric registry
// index for Minecraft Java Edition 1.21.4 (protocol 769).
//
// These IDs are hardcoded for the current vanilla version; they are used:
//   - When sending slot data to the client (resource location → numeric ID).
//   - When receiving Creative Mode Set Item (numeric ID → resource location).
//
// Milestone 13 (data-driven registries) will replace this table by loading
// the real mapping from Minecraft's generated reports/registries.json.
// All entries below are estimates; verify against the generated output if
// a block/item behaves unexpectedly after M13.
//
// Items not present in this table are treated as unplaceable in M10; they
// will be acknowledged but won't create a block in the world.
var itemIDs = map[string]int32{
	// ── Air ──────────────────────────────────────────────────────────────────
	"minecraft:air": 0,

	// ── Stone family ─────────────────────────────────────────────────────────
	"minecraft:stone":            1,
	"minecraft:granite":          2,
	"minecraft:polished_granite": 3,
	"minecraft:diorite":          4,
	"minecraft:polished_diorite": 5,
	"minecraft:andesite":         6,
	"minecraft:polished_andesite": 7,

	// ── Deepslate ─────────────────────────────────────────────────────────────
	"minecraft:deepslate":          8,
	"minecraft:cobbled_deepslate":  9,
	"minecraft:polished_deepslate": 10,
	"minecraft:chiseled_deepslate": 11,

	// ── Ground ────────────────────────────────────────────────────────────────
	"minecraft:grass_block": 13,
	"minecraft:dirt":        14,
	"minecraft:coarse_dirt": 15,
	"minecraft:podzol":      16,
	"minecraft:rooted_dirt": 17,
	"minecraft:mud":         18,

	// ── Cobblestone ───────────────────────────────────────────────────────────
	"minecraft:cobblestone":       21,
	"minecraft:mossy_cobblestone": 22,

	// ── Planks ────────────────────────────────────────────────────────────────
	"minecraft:oak_planks":      26,
	"minecraft:spruce_planks":   27,
	"minecraft:birch_planks":    28,
	"minecraft:jungle_planks":   29,
	"minecraft:acacia_planks":   30,
	"minecraft:cherry_planks":   31,
	"minecraft:dark_oak_planks": 32,
	"minecraft:mangrove_planks": 33,
	"minecraft:bamboo_planks":   34,
	"minecraft:crimson_planks":  35,
	"minecraft:warped_planks":   36,

	// ── Logs ──────────────────────────────────────────────────────────────────
	"minecraft:oak_log":      37,
	"minecraft:spruce_log":   38,
	"minecraft:birch_log":    39,
	"minecraft:jungle_log":   40,
	"minecraft:acacia_log":   41,
	"minecraft:cherry_log":   42,
	"minecraft:dark_oak_log": 43,
	"minecraft:mangrove_log": 44,

	// ── Sand / Gravel ─────────────────────────────────────────────────────────
	"minecraft:sand":      66,
	"minecraft:red_sand":  67,
	"minecraft:gravel":    68,

	// ── Ores ──────────────────────────────────────────────────────────────────
	"minecraft:coal_ore":     69,
	"minecraft:iron_ore":     73,
	"minecraft:gold_ore":     76,
	"minecraft:diamond_ore":  79,
	"minecraft:emerald_ore":  82,
	"minecraft:lapis_ore":    85,

	// ── Glass / Obsidian ──────────────────────────────────────────────────────
	"minecraft:glass":     130,
	"minecraft:obsidian":  131,

	// ── Wool (16 colours) ─────────────────────────────────────────────────────
	"minecraft:white_wool":      140,
	"minecraft:orange_wool":     141,
	"minecraft:magenta_wool":    142,
	"minecraft:light_blue_wool": 143,
	"minecraft:yellow_wool":     144,
	"minecraft:lime_wool":       145,
	"minecraft:pink_wool":       146,
	"minecraft:gray_wool":       147,
	"minecraft:light_gray_wool": 148,
	"minecraft:cyan_wool":       149,
	"minecraft:purple_wool":     150,
	"minecraft:blue_wool":       151,
	"minecraft:brown_wool":      152,
	"minecraft:green_wool":      153,
	"minecraft:red_wool":        154,
	"minecraft:black_wool":      155,

	// ── Concrete ──────────────────────────────────────────────────────────────
	"minecraft:white_concrete":      580,
	"minecraft:orange_concrete":     581,
	"minecraft:magenta_concrete":    582,
	"minecraft:light_blue_concrete": 583,
	"minecraft:yellow_concrete":     584,
	"minecraft:lime_concrete":       585,
	"minecraft:pink_concrete":       586,
	"minecraft:gray_concrete":       587,
	"minecraft:light_gray_concrete": 588,
	"minecraft:cyan_concrete":       589,
	"minecraft:purple_concrete":     590,
	"minecraft:blue_concrete":       591,
	"minecraft:brown_concrete":      592,
	"minecraft:green_concrete":      593,
	"minecraft:red_concrete":        594,
	"minecraft:black_concrete":      595,

	// ── Stone bricks ──────────────────────────────────────────────────────────
	"minecraft:stone_bricks":          174,
	"minecraft:mossy_stone_bricks":    175,
	"minecraft:cracked_stone_bricks":  176,
	"minecraft:chiseled_stone_bricks": 177,

	// ── Bedrock ───────────────────────────────────────────────────────────────
	"minecraft:bedrock": 23,
}

// itemNames is the reverse of itemIDs: numeric item ID → resource location.
// Populated once at init time from itemIDs.
var itemNames map[int32]string

func init() {
	itemNames = make(map[int32]string, len(itemIDs))
	for name, id := range itemIDs {
		itemNames[id] = name
	}
}

// ItemName returns the resource location for a numeric item ID, or "" if
// the item is not in the table.
func ItemName(id int32) string {
	return itemNames[id]
}

// ItemID returns the numeric item ID for a resource location, or -1 if the
// item is not in the table.
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
	// Split "namespace:name" into parts.
	ns, name := "minecraft", itemID
	if i := len("minecraft:"); len(itemID) > i && itemID[:i] == "minecraft:" {
		name = itemID[i:]
	}
	return coreworld.Block{Namespace: ns, Name: name}
}
