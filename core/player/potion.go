package player

import "strings"

type PotionOutcome struct {
	Effects []StatusEffect
	Heal    float32
	Damage  float32
}
type potionContents struct {
	Potion        string         `json:"potion"`
	CustomEffects []StatusEffect `json:"custom_effects"`
}

func PotionOutcomeFor(stack ItemStack) (PotionOutcome, bool) {
	switch stack.ItemID {
	case "minecraft:potion", "minecraft:splash_potion", "minecraft:lingering_potion":
	default:
		return PotionOutcome{}, false
	}
	var contents potionContents
	if !stack.Component("potion_contents", &contents) {
		return PotionOutcome{}, true
	}
	out := basePotionOutcome(strings.TrimPrefix(contents.Potion, "minecraft:"))
	for _, effect := range contents.CustomEffects {
		effect, ok := normalizeStatusEffect(effect)
		if !ok {
			continue
		}
		if applyInstantPotion(&out, effect.ID, effect.Amplifier) {
			continue
		}
		out.Effects = append(out.Effects, effect)
	}
	return out, true
}

func basePotionOutcome(name string) PotionOutcome {
	effect := func(id string, amplifier, duration int32) PotionOutcome {
		return PotionOutcome{Effects: []StatusEffect{{ID: "minecraft:" + id, Amplifier: amplifier, Duration: duration, ShowParticles: true, ShowIcon: true}}}
	}
	switch name {
	case "healing":
		return PotionOutcome{Heal: 4}
	case "strong_healing":
		return PotionOutcome{Heal: 8}
	case "harming":
		return PotionOutcome{Damage: 6}
	case "strong_harming":
		return PotionOutcome{Damage: 12}
	case "night_vision", "invisibility", "fire_resistance", "swiftness", "strength", "water_breathing":
		return effect(potionEffectID(name), 0, 3600)
	case "long_night_vision", "long_invisibility", "long_fire_resistance", "long_swiftness", "long_strength", "long_water_breathing":
		return effect(potionEffectID(strings.TrimPrefix(name, "long_")), 0, 9600)
	case "leaping":
		return effect("jump_boost", 0, 3600)
	case "long_leaping":
		return effect("jump_boost", 0, 9600)
	case "strong_leaping", "strong_swiftness", "strong_strength":
		return effect(potionEffectID(strings.TrimPrefix(name, "strong_")), 1, 1800)
	case "poison", "regeneration":
		return effect(name, 0, 900)
	case "long_poison", "long_regeneration":
		return effect(strings.TrimPrefix(name, "long_"), 0, 1800)
	case "strong_poison":
		return effect("poison", 1, 432)
	case "strong_regeneration":
		return effect("regeneration", 1, 450)
	case "weakness", "slowness":
		return effect(name, 0, 1800)
	case "long_weakness", "long_slowness", "long_slow_falling":
		return effect(strings.TrimPrefix(name, "long_"), 0, 4800)
	case "slow_falling":
		return effect("slow_falling", 0, 1800)
	case "strong_slowness":
		return effect("slowness", 3, 400)
	default:
		return PotionOutcome{}
	}
}

func potionEffectID(name string) string {
	if name == "swiftness" {
		return "speed"
	}
	return name
}

func applyInstantPotion(out *PotionOutcome, id string, amplifier int32) bool {
	switch id {
	case "minecraft:instant_health":
		out.Heal += float32(int32(4) << min(amplifier, 10))
	case "minecraft:instant_damage":
		out.Damage += float32(int32(6) << min(amplifier, 10))
	default:
		return false
	}
	return true
}
