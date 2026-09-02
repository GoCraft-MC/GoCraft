package player

// FoodStatusEffects resolves vanilla food side effects from a 0-99 roll.
func FoodStatusEffects(itemID string, roll int) []StatusEffect {
	effect := func(id string, amplifier, duration int32) StatusEffect {
		return StatusEffect{ID: id, Amplifier: amplifier, Duration: duration, ShowParticles: true, ShowIcon: true}
	}
	switch itemID {
	case "minecraft:rotten_flesh":
		if roll%100 < 80 {
			return []StatusEffect{effect("hunger", 0, 600)}
		}
	case "minecraft:chicken":
		if roll%100 < 30 {
			return []StatusEffect{effect("hunger", 0, 600)}
		}
	case "minecraft:spider_eye":
		return []StatusEffect{effect("poison", 0, 100)}
	case "minecraft:poisonous_potato":
		if roll%100 < 60 {
			return []StatusEffect{effect("poison", 3, 100)}
		}
	case "minecraft:pufferfish":
		return []StatusEffect{
			effect("poison", 3, 1200), effect("hunger", 2, 300), effect("nausea", 1, 300),
		}
	case "minecraft:golden_apple":
		return []StatusEffect{effect("regeneration", 1, 100), effect("absorption", 0, 2400)}
	case "minecraft:enchanted_golden_apple":
		return []StatusEffect{
			effect("regeneration", 4, 600), effect("absorption", 3, 2400),
			effect("resistance", 0, 6000), effect("fire_resistance", 0, 6000),
		}
	}
	return nil
}
