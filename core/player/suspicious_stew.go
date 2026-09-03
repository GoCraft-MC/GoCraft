package player

// SuspiciousStewEffects returns the per-stack effects selected by the flower
// used to craft a suspicious stew.
func SuspiciousStewEffects(stack ItemStack) []StatusEffect {
	if stack.ItemID != "minecraft:suspicious_stew" {
		return nil
	}
	var encoded []StatusEffect
	if !stack.Component("suspicious_stew_effects", &encoded) {
		return nil
	}
	effects := make([]StatusEffect, 0, len(encoded))
	for _, effect := range encoded {
		effect, ok := normalizeStatusEffect(effect)
		if !ok {
			continue
		}
		effect.ShowParticles = true
		effect.ShowIcon = true
		effects = append(effects, effect)
	}
	return effects
}
