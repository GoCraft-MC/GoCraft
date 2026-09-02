package main

import (
	"encoding/json"
	"fmt"
)

func equipmentFrom(raw json.RawMessage) *equipmentProperties {
	if !componentPresent(raw) {
		return nil
	}
	var equipment equipmentProperties
	if err := json.Unmarshal(raw, &equipment); err != nil {
		panic(fmt.Errorf("decode equippable component: %w", err))
	}
	return &equipment
}

func ensureEquipment(equipment *equipmentProperties, slot string) *equipmentProperties {
	if equipment == nil {
		return &equipmentProperties{Slot: slot}
	}
	if equipment.Slot == "" {
		equipment.Slot = slot
	}
	return equipment
}

func attributeModifiers(raw json.RawMessage) []attributeModifier {
	if !componentPresent(raw) {
		return nil
	}
	var wrapped struct {
		Modifiers []attributeModifier `json:"modifiers"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Modifiers != nil {
		return wrapped.Modifiers
	}
	var direct []attributeModifier
	if err := json.Unmarshal(raw, &direct); err != nil {
		panic(fmt.Errorf("decode attribute modifiers: %w", err))
	}
	return direct
}
