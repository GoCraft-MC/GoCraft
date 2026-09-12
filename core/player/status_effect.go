package player

import "strings"

// StatusEffect is an edition-independent active mob effect. Duration is in
// 20 Hz server ticks and Amplifier is zero-based (zero means level I).
type StatusEffect struct {
	ID            string `json:"id"`
	Amplifier     int32  `json:"amplifier,omitempty"`
	Duration      int32  `json:"duration"`
	Ambient       bool   `json:"ambient,omitempty"`
	ShowParticles bool   `json:"show_particles"`
	ShowIcon      bool   `json:"show_icon"`
}

func normalizeStatusEffect(effect StatusEffect) (StatusEffect, bool) {
	effect.ID = strings.TrimSpace(effect.ID)
	if effect.ID != "" && !strings.ContainsRune(effect.ID, ':') {
		effect.ID = "minecraft:" + effect.ID
	}
	if effect.ID == "" || effect.Duration <= 0 {
		return StatusEffect{}, false
	}
	if effect.Amplifier < 0 {
		effect.Amplifier = 0
	}
	return effect, true
}

func statusEffectIndex(effects []StatusEffect, id string) int {
	if !strings.ContainsRune(id, ':') {
		id = "minecraft:" + id
	}
	for index := range effects {
		if effects[index].ID == id {
			return index
		}
	}
	return -1
}
