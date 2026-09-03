package bedrock

import "GoCraft/core/player"

var bedrockPotionNames = [...]string{
	"water", "mundane", "long_mundane", "thick", "awkward",
	"night_vision", "long_night_vision", "invisibility", "long_invisibility",
	"leaping", "long_leaping", "strong_leaping",
	"fire_resistance", "long_fire_resistance",
	"swiftness", "long_swiftness", "strong_swiftness",
	"slowness", "long_slowness",
	"water_breathing", "long_water_breathing",
	"healing", "strong_healing", "harming", "strong_harming",
	"poison", "long_poison", "strong_poison",
	"regeneration", "long_regeneration", "strong_regeneration",
	"strength", "long_strength", "strong_strength",
	"weakness", "long_weakness", "decay",
	"turtle_master", "long_turtle_master", "strong_turtle_master",
	"slow_falling", "long_slow_falling", "strong_slowness",
	"wind_charged", "weaving", "oozing", "infested",
}

func bedrockPotionName(id int16) (string, bool) {
	if id < 0 || int(id) >= len(bedrockPotionNames) {
		return "", false
	}
	return bedrockPotionNames[id], true
}

func bedrockPotionID(stack player.ItemStack) (int16, bool) {
	name, ok := player.PotionName(stack)
	if !ok {
		return 0, false
	}
	if name == "" {
		return 0, true
	}
	for id, candidate := range bedrockPotionNames {
		if candidate == name {
			return int16(id), true
		}
	}
	return 0, false
}

func setBedrockPotionContents(stack *player.ItemStack, id int16) bool {
	name, ok := bedrockPotionName(id)
	if !ok || stack == nil {
		return false
	}
	return stack.SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:" + name,
	}) == nil
}
