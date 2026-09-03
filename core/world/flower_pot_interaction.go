package world

import "strings"

// PottedBlock returns the block placed into an empty flower pot by itemID.
func PottedBlock(itemID string) (Block, bool) {
	name := strings.TrimPrefix(itemID, "minecraft:")
	switch name {
	case "oak_sapling", "spruce_sapling", "birch_sapling", "jungle_sapling", "acacia_sapling",
		"dark_oak_sapling", "mangrove_propagule", "cherry_sapling", "pale_oak_sapling",
		"fern", "dandelion", "poppy", "blue_orchid", "allium", "azure_bluet", "red_tulip",
		"orange_tulip", "white_tulip", "pink_tulip", "oxeye_daisy", "cornflower",
		"lily_of_the_valley", "wither_rose", "red_mushroom", "brown_mushroom", "dead_bush",
		"cactus", "bamboo", "crimson_fungus", "warped_fungus", "crimson_roots", "warped_roots",
		"azalea", "flowering_azalea", "torchflower", "closed_eyeblossom", "open_eyeblossom":
		if name == "azalea" || name == "flowering_azalea" {
			name += "_bush"
		}
		return Block{Namespace: "minecraft", Name: "potted_" + name}, true
	default:
		return Block{}, false
	}
}

// PottedItem returns the item held by a potted vanilla plant.
func PottedItem(block Block) (string, bool) {
	name := block.ResourceLocation()
	if !strings.HasPrefix(name, "minecraft:potted_") {
		return "", false
	}
	item := strings.TrimPrefix(name, "minecraft:potted_")
	item = strings.TrimSuffix(item, "_bush")
	if _, ok := PottedBlock("minecraft:" + item); !ok {
		return "", false
	}
	return "minecraft:" + item, true
}
