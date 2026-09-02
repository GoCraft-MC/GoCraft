package player

import "strings"

func (p *Player) hasStatusEffectLocked(id string) bool {
	return statusEffectIndex(p.StatusEffects, id) >= 0
}

func (p *Player) resistedDamageLocked(amount float32, cause string) float32 {
	index := statusEffectIndex(p.StatusEffects, "minecraft:resistance")
	if index < 0 || strings.Contains(cause, "void") || strings.Contains(cause, "starv") {
		return amount
	}
	reduction := 0.2 * float32(p.StatusEffects[index].Amplifier+1)
	if reduction >= 1 {
		return 0
	}
	return amount * (1 - reduction)
}

func isFireDamageCause(cause string) bool {
	cause = strings.ToLower(cause)
	return strings.Contains(cause, "fire") || strings.Contains(cause, "lava") || strings.Contains(cause, "burn")
}
