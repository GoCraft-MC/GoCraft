package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/player"
	bedrockpacket "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestPlayerStatusEffectsTickInServer(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{11}, "affected", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Health = 10
	p.StatusEffects = []player.StatusEffect{
		{ID: "minecraft:poison", Duration: 25},
		{ID: "minecraft:speed", Duration: 1},
	}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}

	s.tickPlayerStatusEffects()
	health, _, _, _ := p.HealthSnapshot()
	if health != 9 {
		t.Fatalf("health = %v, want 9", health)
	}
	effects := p.StatusEffectsSnapshot()
	if len(effects) != 1 || effects[0].ID != "minecraft:poison" || effects[0].Duration != 24 {
		t.Fatalf("remaining effects = %+v", effects)
	}
}

func TestBedrockEffectTypeCoverage(t *testing.T) {
	tests := map[string]int32{
		"minecraft:speed":           bedrockpacket.EffectSpeed,
		"minecraft:mining_fatigue":  bedrockpacket.EffectMiningFatigue,
		"minecraft:water_breathing": bedrockpacket.EffectWaterBreathing,
		"minecraft:slow_falling":    bedrockpacket.EffectSlowFalling,
		"minecraft:luck":            0,
	}
	for id, want := range tests {
		if got := bedrockEffectType(id); got != want {
			t.Errorf("bedrockEffectType(%q) = %d, want %d", id, got, want)
		}
	}
}
