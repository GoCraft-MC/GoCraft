package player

// AddStatusEffect adds or upgrades an effect and reports the stored value.
func (p *Player) AddStatusEffect(effect StatusEffect) (StatusEffect, bool) {
	effect, ok := normalizeStatusEffect(effect)
	if p == nil || !ok {
		return StatusEffect{}, false
	}
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	index := statusEffectIndex(p.StatusEffects, effect.ID)
	if index >= 0 {
		current := p.StatusEffects[index]
		if current.Amplifier > effect.Amplifier || current.Amplifier == effect.Amplifier && current.Duration >= effect.Duration {
			return current, false
		}
		p.StatusEffects[index] = effect
	} else {
		p.StatusEffects = append(p.StatusEffects, effect)
	}
	if effect.ID == "minecraft:absorption" {
		p.Absorption = max(p.Absorption, float32(4*(effect.Amplifier+1)))
	}
	return effect, true
}

// RemoveStatusEffect removes one effect and returns its former value.
func (p *Player) RemoveStatusEffect(id string) (StatusEffect, bool) {
	if p == nil {
		return StatusEffect{}, false
	}
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	index := statusEffectIndex(p.StatusEffects, id)
	if index < 0 {
		return StatusEffect{}, false
	}
	removed := p.StatusEffects[index]
	p.StatusEffects = append(p.StatusEffects[:index], p.StatusEffects[index+1:]...)
	if removed.ID == "minecraft:absorption" {
		p.Absorption = 0
	}
	return removed, true
}

// ClearStatusEffects removes every active effect and returns the removed set.
func (p *Player) ClearStatusEffects() []StatusEffect {
	if p == nil {
		return nil
	}
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	removed := append([]StatusEffect(nil), p.StatusEffects...)
	p.StatusEffects = nil
	p.Absorption = 0
	return removed
}

func (p *Player) StatusEffectsSnapshot() []StatusEffect {
	if p == nil {
		return nil
	}
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	return append([]StatusEffect(nil), p.StatusEffects...)
}

func (p *Player) StatusEffect(id string) (StatusEffect, bool) {
	if p == nil {
		return StatusEffect{}, false
	}
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	index := statusEffectIndex(p.StatusEffects, id)
	if index < 0 {
		return StatusEffect{}, false
	}
	return p.StatusEffects[index], true
}

// MovementSpeedMultiplier returns the combined vanilla speed/slowness scale.
func (p *Player) MovementSpeedMultiplier() float64 {
	multiplier := 1.0
	if effect, ok := p.StatusEffect("speed"); ok {
		multiplier *= 1 + 0.2*float64(effect.Amplifier+1)
	}
	if effect, ok := p.StatusEffect("slowness"); ok {
		multiplier *= max(0, 1-0.15*float64(effect.Amplifier+1))
	}
	return multiplier
}
