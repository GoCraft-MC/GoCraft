package bedrock

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// EffectType translates a canonical effect resource location to Bedrock's
// protocol identifier. Zero means the effect has no Bedrock equivalent.
func EffectType(id string) int32 {
	switch id {
	case "minecraft:speed":
		return packet.EffectSpeed
	case "minecraft:slowness":
		return packet.EffectSlowness
	case "minecraft:haste":
		return packet.EffectHaste
	case "minecraft:mining_fatigue":
		return packet.EffectMiningFatigue
	case "minecraft:strength":
		return packet.EffectStrength
	case "minecraft:instant_health":
		return packet.EffectInstantHealth
	case "minecraft:instant_damage":
		return packet.EffectInstantDamage
	case "minecraft:jump_boost":
		return packet.EffectJumpBoost
	case "minecraft:nausea":
		return packet.EffectNausea
	case "minecraft:regeneration":
		return packet.EffectRegeneration
	case "minecraft:resistance":
		return packet.EffectResistance
	case "minecraft:fire_resistance":
		return packet.EffectFireResistance
	case "minecraft:water_breathing":
		return packet.EffectWaterBreathing
	case "minecraft:invisibility":
		return packet.EffectInvisibility
	case "minecraft:blindness":
		return packet.EffectBlindness
	case "minecraft:night_vision":
		return packet.EffectNightVision
	case "minecraft:hunger":
		return packet.EffectHunger
	case "minecraft:weakness":
		return packet.EffectWeakness
	case "minecraft:poison":
		return packet.EffectPoison
	case "minecraft:wither":
		return packet.EffectWither
	case "minecraft:health_boost":
		return packet.EffectHealthBoost
	case "minecraft:absorption":
		return packet.EffectAbsorption
	case "minecraft:saturation":
		return packet.EffectSaturation
	case "minecraft:levitation":
		return packet.EffectLevitation
	case "minecraft:conduit_power":
		return packet.EffectConduitPower
	case "minecraft:slow_falling":
		return packet.EffectSlowFalling
	default:
		return 0
	}
}
