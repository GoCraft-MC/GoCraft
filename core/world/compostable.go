package world

import "strings"

// IsCompostable reports items accepted by the vanilla composter interaction.
func IsCompostable(itemID string) bool {
	name := strings.TrimPrefix(itemID, "minecraft:")
	if strings.HasSuffix(name, "_leaves") || strings.HasSuffix(name, "_sapling") ||
		strings.HasSuffix(name, "_seeds") || strings.HasSuffix(name, "_flower") {
		return true
	}
	switch name {
	case "short_grass", "grass", "fern", "seagrass", "kelp", "dried_kelp", "cactus", "sugar_cane",
		"vine", "glow_lichen", "lily_pad", "moss_carpet", "moss_block", "hanging_roots", "mangrove_roots",
		"apple", "melon_slice", "melon", "pumpkin", "carved_pumpkin", "potato", "baked_potato",
		"poisonous_potato", "carrot", "beetroot", "wheat", "bread", "cookie", "cake", "pumpkin_pie",
		"sweet_berries", "glow_berries", "brown_mushroom", "red_mushroom", "mushroom_stem",
		"crimson_fungus", "warped_fungus", "crimson_roots", "warped_roots", "nether_sprouts",
		"weeping_vines", "twisting_vines", "azalea", "flowering_azalea", "big_dripleaf", "small_dripleaf",
		"spore_blossom", "sea_pickle", "hay_block", "dried_kelp_block", "shroomlight":
		return true
	default:
		return false
	}
}

// ComposterAccepts resolves the deterministic success roll for one consumed item.
func ComposterAccepts(x, y, z int, worldAge int64, itemID string) bool {
	chance := uint64(650)
	name := strings.TrimPrefix(itemID, "minecraft:")
	if strings.HasSuffix(name, "_leaves") || strings.HasSuffix(name, "_sapling") || strings.HasSuffix(name, "_seeds") ||
		name == "short_grass" || name == "grass" || name == "fern" {
		chance = 300
	}
	switch name {
	case "cake", "pumpkin_pie":
		chance = 1000
	case "baked_potato", "bread", "cookie", "hay_block", "dried_kelp_block", "shroomlight":
		chance = 850
	}
	hash := uint64(int64(x))*0x9e3779b185ebca87 ^ uint64(int64(y))*0xc2b2ae3d27d4eb4f ^
		uint64(int64(z))*0x165667b19e3779f9 ^ uint64(worldAge) ^ uint64(len(itemID))*0x27d4eb2f165667c5
	hash ^= hash >> 33
	hash *= 0xff51afd7ed558ccd
	hash ^= hash >> 33
	return hash%1000 < chance
}
