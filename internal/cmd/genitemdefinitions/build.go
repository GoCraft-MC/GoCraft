package main

import (
	"encoding/json"
	"fmt"
	"math"
)

func buildDefinition(itemID string, components map[string]json.RawMessage, tags []string) definition {
	result := definition{
		MaxStackSize:  componentInt(components, "minecraft:max_stack_size", 0),
		MaxDurability: componentInt(components, "minecraft:max_damage", 0),
		Rarity:        componentString(components, "minecraft:rarity", "common"),
	}
	if result.MaxStackSize <= 0 {
		panic(fmt.Sprintf("%s has no valid max stack size", itemID))
	}
	applyAttributes(&result, components["minecraft:attribute_modifiers"], components["minecraft:equippable"])
	applyConsumption(itemID, &result, components)
	applyStaticComponents(itemID, &result, components)
	if category := toolCategory(itemID, tags); category != "" {
		result.Tool = toolFrom(itemID, category, components["minecraft:tool"], result.Repair)
	}
	return result
}

func applyAttributes(result *definition, modifiersRaw, equipmentRaw json.RawMessage) {
	equipment := equipmentFrom(equipmentRaw)
	var attackDamage, attackSpeed float64
	var hasAttackDamage, hasAttackSpeed bool
	for _, modifier := range attributeModifiers(modifiersRaw) {
		if modifier.Operation != "add_value" {
			continue
		}
		switch modifier.Type {
		case "minecraft:armor":
			equipment = ensureEquipment(equipment, modifier.Slot)
			equipment.Armor = int(math.Round(modifier.Amount))
		case "minecraft:armor_toughness":
			equipment = ensureEquipment(equipment, modifier.Slot)
			equipment.Toughness = cleanFloat(modifier.Amount)
		case "minecraft:knockback_resistance":
			equipment = ensureEquipment(equipment, modifier.Slot)
			equipment.KnockbackResistance = cleanFloat(modifier.Amount)
		case "minecraft:attack_damage":
			attackDamage, hasAttackDamage = modifier.Amount, true
		case "minecraft:attack_speed":
			attackSpeed, hasAttackSpeed = modifier.Amount, true
		}
	}
	result.Equipment = equipment
	if hasAttackDamage || hasAttackSpeed {
		result.Combat = &combatProperties{
			AttackDamage: cleanFloat(1 + attackDamage),
			AttackSpeed:  cleanFloat(4 + attackSpeed),
		}
	}
}
