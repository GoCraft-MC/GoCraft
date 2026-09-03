package bedrock

import "GoCraft/core/player"

var bedrockStewEffects = [...]player.StatusEffect{
	{ID: "minecraft:night_vision", Duration: 100},
	{ID: "minecraft:jump_boost", Duration: 100},
	{ID: "minecraft:weakness", Duration: 140},
	{ID: "minecraft:blindness", Duration: 120},
	{ID: "minecraft:poison", Duration: 220},
	{ID: "minecraft:saturation", Duration: 6},
	{ID: "minecraft:saturation", Duration: 6},
	{ID: "minecraft:fire_resistance", Duration: 60},
	{ID: "minecraft:regeneration", Duration: 140},
	{ID: "minecraft:wither", Duration: 140},
	{ID: "minecraft:night_vision", Duration: 100},
	{ID: "minecraft:blindness", Duration: 120},
	{ID: "minecraft:nausea", Duration: 140},
}

func setBedrockStewContents(stack *player.ItemStack, variant int16) bool {
	if stack == nil || variant < 0 || int(variant) >= len(bedrockStewEffects) {
		return false
	}
	return stack.SetComponent("suspicious_stew_effects", []player.StatusEffect{
		bedrockStewEffects[variant],
	}) == nil
}

func bedrockStewVariant(stack player.ItemStack) (int16, bool) {
	if stack.ItemID != "minecraft:suspicious_stew" {
		return 0, false
	}
	effects := player.SuspiciousStewEffects(stack)
	if len(effects) == 0 {
		return 0, true
	}
	for variant, candidate := range bedrockStewEffects {
		if effects[0].ID == candidate.ID && effects[0].Duration == candidate.Duration {
			return int16(variant), true
		}
	}
	return 0, false
}
