package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestUntrustingOcelotFleesPlayer(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{9}, "scary", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = s.simulationDimension
	p.Position = spatial.Vec3{X: 20, Y: 64, Z: 26}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	ocelot := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeOcelot, 20, 64, 20)
	s.world.Entities.Add(ocelot)
	ai := s.mobAIFor(ocelot)

	if !s.tickOcelotParity(ocelot, ai, parityState(ocelot)) {
		t.Fatal("untrusting ocelot did not react to the nearby player")
	}
	if ocelot.VZ > 0 {
		t.Fatalf("ocelot fled toward the player (player at +Z): vz=%.3f", ocelot.VZ)
	}
}

func TestOcelotHuntsChicken(t *testing.T) {
	s := newGolemTestServer(t)
	ocelot := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeOcelot, 20, 64, 20)
	chicken := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeChicken, 23, 64, 20)
	s.world.Entities.Add(ocelot)
	s.world.Entities.Add(chicken)
	ai := s.mobAIFor(ocelot)

	if !s.tickOcelotParity(ocelot, ai, parityState(ocelot)) {
		t.Fatal("ocelot did not hunt the nearby chicken")
	}
}

func TestTrustingOcelotDoesNotFleePlayer(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{9}, "friend", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = s.simulationDimension
	p.Position = spatial.Vec3{X: 20, Y: 64, Z: 26}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	ocelot := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeOcelot, 20, 64, 20)
	ocelot.Trusting = true
	s.world.Entities.Add(ocelot)
	ai := s.mobAIFor(ocelot)

	// A trusting ocelot no longer flees; with no prey around it simply idles.
	s.tickOcelotParity(ocelot, ai, parityState(ocelot))
	if ocelot.VZ < 0 {
		t.Fatalf("trusting ocelot fled from the player: vz=%.3f", ocelot.VZ)
	}
}
