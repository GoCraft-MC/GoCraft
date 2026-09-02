package main

import (
	"encoding/json"
	"math"
)

func applyConsumption(itemID string, result *definition, components map[string]json.RawMessage) {
	if raw := components["minecraft:food"]; componentPresent(raw) {
		var food struct {
			Nutrition    int32   `json:"nutrition"`
			Saturation   float32 `json:"saturation"`
			AlwaysEdible bool    `json:"can_always_eat"`
		}
		mustUnmarshalComponent(itemID, "food", raw, &food)
		properties := foodProperties(food)
		properties.Saturation = cleanFloat(float64(properties.Saturation))
		result.Food = &properties
	}
	raw := components["minecraft:consumable"]
	if !componentPresent(raw) {
		return
	}
	var consumable struct {
		ConsumeSeconds float64 `json:"consume_seconds"`
		Animation      string  `json:"animation"`
	}
	mustUnmarshalComponent(itemID, "consumable", raw, &consumable)
	if consumable.ConsumeSeconds == 0 {
		consumable.ConsumeSeconds = 1.6
	}
	result.Consumable = &consumableProperties{
		UseDurationTicks: int(math.Round(consumable.ConsumeSeconds * 20)),
		Animation:        consumable.Animation,
	}
	if raw := components["minecraft:use_remainder"]; componentPresent(raw) {
		var remainder struct {
			ID string `json:"id"`
		}
		mustUnmarshalComponent(itemID, "use_remainder", raw, &remainder)
		result.Consumable.Remainder = remainder.ID
	}
}
