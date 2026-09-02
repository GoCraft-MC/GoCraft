package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/player"
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
