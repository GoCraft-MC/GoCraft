package player

import (
	"sort"
	"strings"
)

type Enchantment struct {
	ID, SupportedItems, ExclusiveSet string
	MaxLevel                         int
}

var enchantments = map[string]Enchantment{
	"minecraft:aqua_affinity":         {"minecraft:aqua_affinity", "head_armor", "", 1},
	"minecraft:bane_of_arthropods":    {"minecraft:bane_of_arthropods", "weapon", "damage", 5},
	"minecraft:binding_curse":         {"minecraft:binding_curse", "equippable", "", 1},
	"minecraft:blast_protection":      {"minecraft:blast_protection", "armor", "armor", 4},
	"minecraft:breach":                {"minecraft:breach", "mace", "damage", 4},
	"minecraft:channeling":            {"minecraft:channeling", "trident", "", 1},
	"minecraft:density":               {"minecraft:density", "mace", "damage", 5},
	"minecraft:depth_strider":         {"minecraft:depth_strider", "foot_armor", "boots", 3},
	"minecraft:efficiency":            {"minecraft:efficiency", "mining", "", 5},
	"minecraft:feather_falling":       {"minecraft:feather_falling", "foot_armor", "", 4},
	"minecraft:fire_aspect":           {"minecraft:fire_aspect", "fire_aspect", "", 2},
	"minecraft:fire_protection":       {"minecraft:fire_protection", "armor", "armor", 4},
	"minecraft:flame":                 {"minecraft:flame", "bow", "", 1},
	"minecraft:fortune":               {"minecraft:fortune", "mining_loot", "mining", 3},
	"minecraft:frost_walker":          {"minecraft:frost_walker", "foot_armor", "boots", 2},
	"minecraft:impaling":              {"minecraft:impaling", "trident", "damage", 5},
	"minecraft:infinity":              {"minecraft:infinity", "bow", "bow", 1},
	"minecraft:knockback":             {"minecraft:knockback", "sword", "", 2},
	"minecraft:looting":               {"minecraft:looting", "sword", "", 3},
	"minecraft:loyalty":               {"minecraft:loyalty", "trident", "", 3},
	"minecraft:luck_of_the_sea":       {"minecraft:luck_of_the_sea", "fishing", "", 3},
	"minecraft:lure":                  {"minecraft:lure", "fishing", "", 3},
	"minecraft:mending":               {"minecraft:mending", "durability", "", 1},
	"minecraft:multishot":             {"minecraft:multishot", "crossbow", "crossbow", 1},
	"minecraft:piercing":              {"minecraft:piercing", "crossbow", "crossbow", 4},
	"minecraft:power":                 {"minecraft:power", "bow", "", 5},
	"minecraft:projectile_protection": {"minecraft:projectile_protection", "armor", "armor", 4},
	"minecraft:protection":            {"minecraft:protection", "armor", "armor", 4},
	"minecraft:punch":                 {"minecraft:punch", "bow", "", 2},
	"minecraft:quick_charge":          {"minecraft:quick_charge", "crossbow", "", 3},
	"minecraft:respiration":           {"minecraft:respiration", "head_armor", "", 3},
	"minecraft:riptide":               {"minecraft:riptide", "trident", "riptide", 3},
	"minecraft:sharpness":             {"minecraft:sharpness", "sharp_weapon", "damage", 5},
	"minecraft:silk_touch":            {"minecraft:silk_touch", "mining_loot", "mining", 1},
	"minecraft:smite":                 {"minecraft:smite", "weapon", "damage", 5},
	"minecraft:soul_speed":            {"minecraft:soul_speed", "foot_armor", "", 3},
	"minecraft:sweeping_edge":         {"minecraft:sweeping_edge", "sword", "", 3},
	"minecraft:swift_sneak":           {"minecraft:swift_sneak", "leg_armor", "", 3},
	"minecraft:thorns":                {"minecraft:thorns", "armor", "", 3},
	"minecraft:unbreaking":            {"minecraft:unbreaking", "durability", "", 3},
	"minecraft:vanishing_curse":       {"minecraft:vanishing_curse", "vanishing", "", 1},
	"minecraft:wind_burst":            {"minecraft:wind_burst", "mace", "", 3},
}

func EnchantmentByID(id string) (Enchantment, bool) {
	if !strings.Contains(id, ":") {
		id = "minecraft:" + id
	}
	value, ok := enchantments[id]
	return value, ok
}

func EnchantmentIDs() []string {
	ids := make([]string, 0, len(enchantments))
	for id := range enchantments {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
