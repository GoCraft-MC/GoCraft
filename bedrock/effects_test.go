package bedrock

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestEffectType(t *testing.T) {
	tests := map[string]int32{
		"minecraft:speed":           packet.EffectSpeed,
		"minecraft:mining_fatigue":  packet.EffectMiningFatigue,
		"minecraft:water_breathing": packet.EffectWaterBreathing,
		"minecraft:slow_falling":    packet.EffectSlowFalling,
		"minecraft:luck":            0,
	}
	for id, want := range tests {
		if got := EffectType(id); got != want {
			t.Errorf("EffectType(%q) = %d, want %d", id, got, want)
		}
	}
}
