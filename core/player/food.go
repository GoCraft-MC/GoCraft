package player

import (
	"GoCraft/core/itemregistry"
	"time"
)

// FoodValue returns vanilla nutrition and saturation modifier values for
// common edible items. Unknown and non-food items return ok=false.
func FoodValue(itemID string) (nutrition int32, saturationModifier float32, ok bool) {
	definition, found := itemregistry.Lookup(itemID)
	if !found || definition.Food == nil {
		return 0, 0, false
	}
	food := definition.Food
	return food.Nutrition, food.SaturationModifier(), true
}

// FoodUseDuration returns the vanilla time required to finish eating an item.
// Unknown items retain the historical 32-tick fallback.
func FoodUseDuration(itemID string) time.Duration {
	definition, ok := itemregistry.Lookup(itemID)
	if ok && definition.Consumable != nil {
		return time.Duration(definition.Consumable.UseDurationTicks) * 50 * time.Millisecond
	}
	return 1600 * time.Millisecond
}

// CanAlwaysEat reports foods whose vanilla component permits use with a full
// hunger bar. Creative players are handled separately by the server.
func CanAlwaysEat(itemID string) bool {
	definition, ok := itemregistry.Lookup(itemID)
	return ok && definition.Food != nil && definition.Food.AlwaysEdible
}

// IsConsumable reports items with a vanilla timed use action, including
// drinks such as milk and potions that do not restore hunger.
func IsConsumable(itemID string) bool {
	definition, ok := itemregistry.Lookup(itemID)
	return ok && definition.Consumable != nil
}

// FoodRemainder returns the container left behind after a food item is eaten.
func FoodRemainder(itemID string) string {
	definition, ok := itemregistry.Lookup(itemID)
	if !ok || definition.Consumable == nil {
		return ""
	}
	return definition.Consumable.Remainder
}
