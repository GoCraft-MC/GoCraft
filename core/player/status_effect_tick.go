package player

// StatusEffectTick describes authoritative work produced by one effect tick.
// The server applies damage so normal armour-independent death/totem handling
// and adapter feedback remain on the existing path.
type StatusEffectTick struct {
	Expired    []StatusEffect
	Heal       float32
	Damage     float32
	Exhaustion float32
	CanKill    bool
}

// TickStatusEffects advances durations and periodic vanilla effect actions.
func (p *Player) TickStatusEffects() StatusEffectTick {
	var result StatusEffectTick
	if p == nil {
		return result
	}
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	kept := p.StatusEffects[:0]
	for _, effect := range p.StatusEffects {
		interval := 0
		switch effect.ID {
		case "minecraft:regeneration":
			interval = shiftedEffectInterval(50, effect.Amplifier)
			if effect.Duration%int32(interval) == 0 && p.Health+result.Heal < p.MaxHealth {
				result.Heal++
			}
		case "minecraft:poison":
			interval = shiftedEffectInterval(25, effect.Amplifier)
			if effect.Duration%int32(interval) == 0 && p.Health-result.Damage > 1 {
				result.Damage++
			}
		case "minecraft:wither":
			interval = shiftedEffectInterval(40, effect.Amplifier)
			if effect.Duration%int32(interval) == 0 {
				result.Damage++
				result.CanKill = true
			}
		case "minecraft:hunger":
			result.Exhaustion += 0.005 * float32(effect.Amplifier+1)
		}
		effect.Duration--
		if effect.Duration <= 0 {
			result.Expired = append(result.Expired, effect)
			if effect.ID == "minecraft:absorption" {
				p.Absorption = 0
			}
			continue
		}
		kept = append(kept, effect)
	}
	p.StatusEffects = kept
	return result
}

func shiftedEffectInterval(base int, amplifier int32) int {
	if amplifier >= 31 {
		return 1
	}
	interval := base >> amplifier
	if interval < 1 {
		return 1
	}
	return interval
}
