package itemregistry

import (
	"fmt"
	"strings"
)

func validateDefinition(definition *Definition) error {
	if definition.ID == "" || !strings.Contains(definition.ID, ":") {
		return fmt.Errorf("itemregistry: invalid item ID %q", definition.ID)
	}
	if definition.MaxStackSize <= 0 {
		return fmt.Errorf("itemregistry: %s has invalid max stack size %d", definition.ID, definition.MaxStackSize)
	}
	if definition.MaxDurability < 0 {
		return fmt.Errorf("itemregistry: %s has negative durability", definition.ID)
	}
	if definition.Equipment != nil && definition.Equipment.Slot == "" {
		return fmt.Errorf("itemregistry: %s has equipment attributes without a slot", definition.ID)
	}
	if definition.Food != nil && definition.Food.Nutrition <= 0 {
		return fmt.Errorf("itemregistry: %s has invalid food nutrition", definition.ID)
	}
	if definition.Consumable != nil && definition.Consumable.UseDurationTicks <= 0 {
		return fmt.Errorf("itemregistry: %s has invalid use duration", definition.ID)
	}
	if definition.Tool != nil && definition.Tool.BlockDamageCost <= 0 {
		return fmt.Errorf("itemregistry: %s has invalid block damage cost", definition.ID)
	}
	return nil
}
