package player

import "strings"

func (e Enchantment) Supports(itemID string) bool {
	switch e.SupportedItems {
	case "armor":
		return armorSlot(itemID) != ""
	case "foot_armor", "leg_armor", "chest_armor", "head_armor":
		return armorSlot(itemID) == strings.TrimSuffix(e.SupportedItems, "_armor")
	case "bow", "crossbow", "mace", "trident":
		return itemID == "minecraft:"+e.SupportedItems
	case "fishing":
		return itemID == "minecraft:fishing_rod"
	case "sword":
		return materialTool(itemID, "sword")
	case "sharp_weapon":
		return materialTool(itemID, "sword", "axe")
	case "weapon":
		return materialTool(itemID, "sword", "axe") || itemID == "minecraft:mace"
	case "fire_aspect":
		return materialTool(itemID, "sword") || itemID == "minecraft:mace"
	case "mining":
		return materialTool(itemID, "axe", "pickaxe", "shovel", "hoe") || itemID == "minecraft:shears"
	case "mining_loot":
		return materialTool(itemID, "axe", "pickaxe", "shovel", "hoe")
	case "durability":
		return enchantableDurability(itemID)
	case "equippable":
		return armorSlot(itemID) != "" || itemID == "minecraft:elytra" || isSkull(itemID) || itemID == "minecraft:carved_pumpkin"
	case "vanishing":
		return enchantableDurability(itemID) || itemID == "minecraft:compass" || isSkull(itemID) || itemID == "minecraft:carved_pumpkin"
	}
	return false
}

func armorSlot(itemID string) string {
	if itemID == "minecraft:turtle_helmet" {
		return "head"
	}
	for _, material := range []string{"leather", "chainmail", "golden", "iron", "diamond", "netherite"} {
		prefix := "minecraft:" + material + "_"
		if !strings.HasPrefix(itemID, prefix) {
			continue
		}
		switch strings.TrimPrefix(itemID, prefix) {
		case "boots":
			return "foot"
		case "leggings":
			return "leg"
		case "chestplate":
			return "chest"
		case "helmet":
			return "head"
		}
	}
	return ""
}

func materialTool(itemID string, kinds ...string) bool {
	for _, material := range []string{"wooden", "stone", "golden", "iron", "diamond", "netherite"} {
		for _, kind := range kinds {
			if itemID == "minecraft:"+material+"_"+kind {
				return true
			}
		}
	}
	return false
}

func enchantableDurability(itemID string) bool {
	return armorSlot(itemID) != "" || materialTool(itemID, "sword", "axe", "pickaxe", "shovel", "hoe") ||
		itemID == "minecraft:elytra" || itemID == "minecraft:shield" || itemID == "minecraft:bow" ||
		itemID == "minecraft:crossbow" || itemID == "minecraft:trident" || itemID == "minecraft:flint_and_steel" ||
		itemID == "minecraft:shears" || itemID == "minecraft:brush" || itemID == "minecraft:fishing_rod" ||
		itemID == "minecraft:carrot_on_a_stick" || itemID == "minecraft:warped_fungus_on_a_stick" || itemID == "minecraft:mace"
}

func isSkull(itemID string) bool {
	switch itemID {
	case "minecraft:player_head", "minecraft:creeper_head", "minecraft:zombie_head", "minecraft:skeleton_skull",
		"minecraft:wither_skeleton_skull", "minecraft:dragon_head", "minecraft:piglin_head":
		return true
	}
	return false
}
