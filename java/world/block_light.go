package world

import (
	"strings"

	coreworld "GoCraft/core/world"
)

func blockLightPasses(block coreworld.Block) bool {
	return isSkyTransparent(block.ResourceLocation()) || blockLightEmission(block) != 0
}

// BlockLightChanged reports whether a canonical mutation can alter propagated
// light and therefore requires Java light-section updates.
func BlockLightChanged(previous, current coreworld.Block) bool {
	return blockLightEmission(previous) != blockLightEmission(current) ||
		blockLightPasses(previous) != blockLightPasses(current)
}

func blockLightEmission(block coreworld.Block) byte {
	name := block.ResourceLocation()
	switch name {
	case "minecraft:torch", "minecraft:wall_torch":
		return 14
	case "minecraft:soul_torch", "minecraft:soul_wall_torch", "minecraft:soul_lantern", "minecraft:soul_fire":
		return 10
	case "minecraft:glowstone", "minecraft:sea_lantern", "minecraft:shroomlight", "minecraft:jack_o_lantern",
		"minecraft:lantern", "minecraft:end_rod", "minecraft:beacon", "minecraft:fire", "minecraft:lava":
		return 15
	case "minecraft:magma_block":
		return 3
	case "minecraft:glow_lichen":
		return 7
	case "minecraft:redstone_ore", "minecraft:deepslate_redstone_ore":
		if block.Properties["lit"] == "true" {
			return 9
		}
	case "minecraft:redstone_torch", "minecraft:redstone_wall_torch":
		if block.Properties["lit"] != "false" {
			return 7
		}
	case "minecraft:furnace", "minecraft:blast_furnace", "minecraft:smoker":
		if block.Properties["lit"] == "true" {
			return 13
		}
	case "minecraft:campfire":
		if block.Properties["lit"] == "true" {
			return 15
		}
	case "minecraft:soul_campfire":
		if block.Properties["lit"] == "true" {
			return 10
		}
	}
	if (name == "minecraft:candle" || name == "minecraft:candle_cake" ||
		strings.HasSuffix(name, "_candle") || strings.HasSuffix(name, "_candle_cake")) &&
		block.Properties["lit"] == "true" {
		count := 1
		if value := block.Properties["candles"]; value == "2" {
			count = 2
		} else if value == "3" {
			count = 3
		} else if value == "4" {
			count = 4
		}
		return byte(count * 3)
	}
	return 0
}
