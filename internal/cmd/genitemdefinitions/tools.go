package main

import "encoding/json"

func toolFrom(itemID, category string, raw json.RawMessage, repair *repairProperties) *toolProperties {
	tool := &toolProperties{Category: category, Tier: toolTier(repair), BlockDamageCost: 1}
	if !componentPresent(raw) {
		return tool
	}
	var component struct {
		DamagePerBlock int `json:"damage_per_block"`
		Rules          []struct {
			Blocks json.RawMessage `json:"blocks"`
			Speed  float32         `json:"speed"`
		} `json:"rules"`
	}
	mustUnmarshalComponent(itemID, "tool", raw, &component)
	if component.DamagePerBlock > 0 {
		tool.BlockDamageCost = component.DamagePerBlock
	}
	for _, rule := range component.Rules {
		for _, blockSet := range stringValues(rule.Blocks) {
			if blockSet == "#minecraft:mineable/"+category {
				tool.MiningSpeed = cleanFloat(float64(rule.Speed))
			}
		}
	}
	return tool
}

func toolCategory(itemID string, tags []string) string {
	for _, tag := range tags {
		switch tag {
		case "minecraft:swords":
			return "sword"
		case "minecraft:axes":
			return "axe"
		case "minecraft:pickaxes":
			return "pickaxe"
		case "minecraft:shovels":
			return "shovel"
		case "minecraft:hoes":
			return "hoe"
		case "minecraft:enchantable/fishing":
			return "fishing_rod"
		}
	}
	switch itemID {
	case "minecraft:shears":
		return "shears"
	case "minecraft:brush":
		return "brush"
	case "minecraft:flint_and_steel":
		return "flint_and_steel"
	case "minecraft:carrot_on_a_stick":
		return "carrot_on_a_stick"
	case "minecraft:warped_fungus_on_a_stick":
		return "warped_fungus_on_a_stick"
	case "minecraft:trident":
		return "trident"
	case "minecraft:mace":
		return "mace"
	}
	return ""
}

func toolTier(repair *repairProperties) string {
	if repair == nil {
		return ""
	}
	tiers := map[string]string{
		"#minecraft:wooden_tool_materials":    "wooden",
		"#minecraft:stone_tool_materials":     "stone",
		"#minecraft:iron_tool_materials":      "iron",
		"#minecraft:gold_tool_materials":      "golden",
		"#minecraft:diamond_tool_materials":   "diamond",
		"#minecraft:netherite_tool_materials": "netherite",
	}
	return tiers[repair.Ingredient]
}
