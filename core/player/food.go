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
// Most foods take 32 ticks; dried kelp is deliberately twice as fast.
func FoodUseDuration(itemID string) time.Duration {
	if itemID == "minecraft:dried_kelp" {
		return 800 * time.Millisecond
	}
	return 1600 * time.Millisecond
}

// CanAlwaysEat reports foods whose vanilla component permits use with a full
// hunger bar. Creative players are handled separately by the server.
func CanAlwaysEat(itemID string) bool {
	switch itemID {
	case "minecraft:golden_apple", "minecraft:enchanted_golden_apple", "minecraft:chorus_fruit":
		return true
	default:
		return false
	}
}

// FoodRemainder returns the container left behind after a food item is eaten.
func FoodRemainder(itemID string) string {
	switch itemID {
	case "minecraft:mushroom_stew", "minecraft:rabbit_stew", "minecraft:beetroot_soup", "minecraft:suspicious_stew":
		return "minecraft:bowl"
	case "minecraft:honey_bottle":
		return "minecraft:glass_bottle"
	default:
		return ""
	}
}
