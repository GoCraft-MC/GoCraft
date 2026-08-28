package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestBreathingFollowsPlayerEyeBlock(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{1}, "diver", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(0, 65, 0, coreworld.MakeFluid("minecraft:water", 0))
	s := &Server{game: g, world: w, sessions: session.NewManager()}

	s.tickPlayerBreathing()
	if air := p.AirSupplySnapshot(); air != 299 {
		t.Fatalf("underwater air = %d, want 299", air)
	}
	p.AirSupply, p.DrowningTicks = 0, 19
	s.tickPlayerBreathing()
	if health, _, _, _ := p.HealthSnapshot(); health != 18 {
		t.Fatalf("health after drowning tick = %v, want 18", health)
	}
	w.SetBlock(0, 65, 0, coreworld.Air)
	s.tickPlayerBreathing()
	if air := p.AirSupplySnapshot(); air != 4 {
		t.Fatalf("recovered air = %d, want 4", air)
	}
}
