package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/session"
)

func TestNewExperienceOrbSurvivesFirstBedrockSync(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{1}, "bedrock", player.ClientEditionBedrock)
	p.Position = spatial.Vec3{X: 1, Y: 64, Z: 1}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, sessions: session.NewManager(), worldAge: 10}
	orb := corentity.New(2, [16]byte{2}, corentity.TypeExperienceOrb, 1, 64.5, 1)
	orb.ExperienceAmount = 3
	orb.AgeTicks = 1

	if s.tryPickupExperienceOrb(orb, dimensionOverworld) {
		t.Fatal("new orb was collected before Bedrock could spawn it")
	}
	orb.AgeTicks = 2
	if !s.tryPickupExperienceOrb(orb, dimensionOverworld) {
		t.Fatal("orb was not collected after its first sync")
	}
	if _, total, _ := p.ExperienceSnapshot(); total != 3 {
		t.Fatalf("experience total = %d, want 3", total)
	}
}
