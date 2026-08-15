package player

import "time"

// FoodValue returns vanilla nutrition and saturation modifier values for
// common edible items. Unknown and non-food items return ok=false.
func FoodValue(itemID string) (nutrition int32, saturationModifier float32, ok bool) {
	switch itemID {
	case "minecraft:apple", "minecraft:chorus_fruit":
		return 4, 0.3, true
	case "minecraft:baked_potato":
		return 5, 0.6, true
	case "minecraft:beef", "minecraft:porkchop", "minecraft:rabbit":
		return 3, 0.3, true
	case "minecraft:beetroot", "minecraft:dried_kelp", "minecraft:melon_slice", "minecraft:sweet_berries", "minecraft:glow_berries":
		return 2, 0.1, true
	case "minecraft:beetroot_soup", "minecraft:cooked_chicken", "minecraft:cooked_mutton", "minecraft:cooked_rabbit", "minecraft:mushroom_stew", "minecraft:suspicious_stew":
		return 6, 0.6, true
	case "minecraft:bread":
		return 5, 0.6, true
	case "minecraft:carrot":
		return 3, 0.6, true
	case "minecraft:chicken", "minecraft:mutton", "minecraft:rotten_flesh":
		return 2, 0.3, true
	case "minecraft:cod", "minecraft:salmon":
		return 2, 0.1, true
	case "minecraft:cooked_beef", "minecraft:cooked_porkchop":
		return 8, 0.8, true
	case "minecraft:cooked_cod":
		return 5, 0.6, true
	case "minecraft:cooked_salmon":
		return 6, 0.8, true
	case "minecraft:cookie":
		return 2, 0.1, true
	case "minecraft:golden_apple", "minecraft:enchanted_golden_apple":
		return 4, 1.2, true
	case "minecraft:golden_carrot":
		return 6, 1.2, true
	case "minecraft:honey_bottle":
		return 6, 0.1, true
	case "minecraft:poisonous_potato", "minecraft:spider_eye":
		return 2, 0.3, true
	case "minecraft:potato":
		return 1, 0.3, true
	case "minecraft:pufferfish", "minecraft:tropical_fish":
		return 1, 0.1, true
	case "minecraft:pumpkin_pie":
		return 8, 0.3, true
	case "minecraft:rabbit_stew":
		return 10, 0.6, true
	}
	return 0, 0, false
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
