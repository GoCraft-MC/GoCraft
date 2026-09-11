package world

import "GoCraft/core/player"

// HarvestBeehive resolves a full hive harvest. The caller applies tool wear or
// consumes the input container, then publishes the returned output stack.
func HarvestBeehive(block Block, itemID string) (Block, player.ItemStack, bool) {
	name := block.ResourceLocation()
	if (name != "minecraft:beehive" && name != "minecraft:bee_nest") || block.Properties["honey_level"] != "5" {
		return Block{}, player.ItemStack{}, false
	}
	var output player.ItemStack
	switch itemID {
	case "minecraft:shears":
		output = player.ItemStack{ItemID: "minecraft:honeycomb", Count: 3}
	case "minecraft:glass_bottle":
		output = player.ItemStack{ItemID: "minecraft:honey_bottle", Count: 1}
	default:
		return Block{}, player.ItemStack{}, false
	}
	properties := make(map[string]string, len(block.Properties))
	for key, value := range block.Properties {
		properties[key] = value
	}
	properties["honey_level"] = "0"
	block.Properties = properties
	return block, output, true
}
