package blockloot

import "strings"

// Vanilla reuses several standing-block loot tables for runtime-only wall
// variants. The embedded data is keyed by loot-table name, so register the
// block aliases explicitly for the canonical Java and Bedrock break paths.
func init() {
	db := data()
	snapshot := make(map[string]map[string]any, len(db.tables))
	for name, table := range db.tables {
		snapshot[name] = table
	}
	for name, table := range snapshot {
		switch {
		case strings.HasSuffix(name, "_hanging_sign"):
			aliasLootTable(db, strings.TrimSuffix(name, "_hanging_sign")+"_wall_hanging_sign", table)
		case strings.HasSuffix(name, "_sign"):
			aliasLootTable(db, strings.TrimSuffix(name, "_sign")+"_wall_sign", table)
		case strings.HasSuffix(name, "_banner"):
			aliasLootTable(db, strings.TrimSuffix(name, "_banner")+"_wall_banner", table)
		case strings.HasSuffix(name, "_coral_fan"):
			aliasLootTable(db, strings.TrimSuffix(name, "_coral_fan")+"_coral_wall_fan", table)
		}
	}
	for wall, standing := range map[string]string{
		"minecraft:skeleton_wall_skull":        "minecraft:skeleton_skull",
		"minecraft:wither_skeleton_wall_skull": "minecraft:wither_skeleton_skull",
		"minecraft:zombie_wall_head":           "minecraft:zombie_head",
		"minecraft:player_wall_head":           "minecraft:player_head",
		"minecraft:creeper_wall_head":          "minecraft:creeper_head",
		"minecraft:dragon_wall_head":           "minecraft:dragon_head",
		"minecraft:piglin_wall_head":           "minecraft:piglin_head",
	} {
		if table, ok := db.tables[standing]; ok {
			aliasLootTable(db, wall, table)
		}
	}
}

func aliasLootTable(db *database, alias string, table map[string]any) {
	if _, exists := db.tables[alias]; !exists {
		db.tables[alias] = table
	}
}
