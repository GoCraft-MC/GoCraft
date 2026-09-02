package main

import "encoding/json"

func applyStaticComponents(itemID string, result *definition, components map[string]json.RawMessage) {
	if raw := components["minecraft:enchantable"]; componentPresent(raw) {
		var enchantable struct {
			Value int `json:"value"`
		}
		mustUnmarshalComponent(itemID, "enchantable", raw, &enchantable)
		result.Enchantability = enchantable.Value
	}
	if raw := components["minecraft:repairable"]; componentPresent(raw) {
		var repairable struct {
			Items json.RawMessage `json:"items"`
		}
		mustUnmarshalComponent(itemID, "repairable", raw, &repairable)
		if ingredients := stringValues(repairable.Items); len(ingredients) == 1 {
			result.Repair = &repairProperties{Ingredient: canonicalIngredient(ingredients[0])}
		}
	}
	raw := components["minecraft:damage_resistant"]
	if !componentPresent(raw) {
		return
	}
	var resistant struct {
		Types json.RawMessage `json:"types"`
	}
	mustUnmarshalComponent(itemID, "damage_resistant", raw, &resistant)
	for _, damageType := range stringValues(resistant.Types) {
		if damageType == "#minecraft:is_fire" || damageType == "minecraft:is_fire" {
			result.FireResistant = true
		}
	}
}
