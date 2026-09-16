package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestUntamedCatFleesNearbyPlayer(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{9}, "scary", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = s.simulationDimension
	p.Position = spatial.Vec3{X: 26, Y: 64, Z: 20}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	cat := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeCat, 20, 64, 20)
	s.world.Entities.Add(cat)
	ai := s.mobAIFor(cat)

	if !s.tickCatParity(cat, ai, parityState(cat)) {
		t.Fatal("untamed cat did not react to the nearby player")
	}
	if cat.VX > 0 {
		t.Fatalf("cat fled toward the player (player at +X): vx=%.3f", cat.VX)
	}
}

func TestUntamedCatHuntsRabbit(t *testing.T) {
	s := newGolemTestServer(t)
	cat := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeCat, 20, 64, 20)
	rabbit := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeRabbit, 23, 64, 20)
	s.world.Entities.Add(cat)
	s.world.Entities.Add(rabbit)
	ai := s.mobAIFor(cat)

	if !s.tickCatParity(cat, ai, parityState(cat)) {
		t.Fatal("untamed cat did not hunt the nearby rabbit")
	}
}

func TestTamedCatDoesNotFleePlayer(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{9}, "friend", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = s.simulationDimension
	p.Position = spatial.Vec3{X: 26, Y: 64, Z: 20}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	cat := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeCat, 20, 64, 20)
	cat.Tamed = true
	s.world.Entities.Add(cat)
	ai := s.mobAIFor(cat)

	// A tamed cat routes to owner-follow, not the skittish flee; with no tame
	// owner online it simply idles instead of sprinting away from the player.
	s.tickCatParity(cat, ai, parityState(cat))
	if cat.VX < 0 {
		t.Fatalf("tamed cat fled from the player: vx=%.3f", cat.VX)
	}
}
