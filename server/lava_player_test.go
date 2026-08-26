package server

import (
	"testing"
	"time"

	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestStationaryPlayersKeepTakingLavaDamage(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, coreworld.MakeFluid("minecraft:lava", 0))
	g := game.New()
	p := player.New([16]byte{1}, "stationary", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, world: w, sessions: session.NewManager()}

	s.tickStationaryLavaDamage()
	if health, _, _, _ := p.HealthSnapshot(); health != 16 {
		t.Fatalf("first stationary lava damage health=%v", health)
	}
	s.tickStationaryLavaDamage()
	if health, _, _, _ := p.HealthSnapshot(); health != 16 {
		t.Fatalf("lava ignored hurt cooldown: health=%v", health)
	}
	p.LastEnvironmentDamage = time.Now().Add(-600 * time.Millisecond)
	s.tickStationaryLavaDamage()
	if health, _, _, _ := p.HealthSnapshot(); health != 12 {
		t.Fatalf("second stationary lava damage health=%v", health)
	}
}
