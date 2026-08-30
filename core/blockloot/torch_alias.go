package blockloot

// Wall torch blocks use the same vanilla loot tables as their standing item
// variants. Mojang does not expose separate wall-torch loot-table keys, so keep
// the block aliases in the shared loot database used by both protocol adapters.
func init() {
	db := data()
	for wallBlock, standingBlock := range map[string]string{
		"minecraft:wall_torch":          "minecraft:torch",
		"minecraft:soul_wall_torch":     "minecraft:soul_torch",
		"minecraft:redstone_wall_torch": "minecraft:redstone_torch",
	} {
		if table, ok := db.tables[standingBlock]; ok {
			db.tables[wallBlock] = table
		}
	}
}
