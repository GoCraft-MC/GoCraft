package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func newBeeTestAggressor(t *testing.T, s *Server, beeID int32) *player.Player {
	t.Helper()
	p := player.New([16]byte{9}, "victim", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = s.simulationDimension
	p.MaxHealth, p.Health = 20, 20
	p.Position = spatial.Vec3{X: 20, Y: 64, Z: 20}
	p.LastAttackedEntityID = beeID
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAngryBeeStingsAggressorAndLosesStinger(t *testing.T) {
	s := newGolemTestServer(t)
	bee := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeBee, 21, 64, 20)
	s.world.Entities.Add(bee)
	p := newBeeTestAggressor(t, s, bee.EntityID)

	ai := s.mobAIFor(bee)
	state := parityState(bee)
	state.angerTicks = 400

	if !s.tickBeeParity(bee, ai, state) {
		t.Fatal("angry bee did not act on its aggressor")
	}
	if !state.beeHasStung {
		t.Fatal("bee did not register a sting")
	}
	if p.Health >= 20 {
		t.Fatalf("bee sting dealt no damage: health=%.1f", p.Health)
	}
	if _, ok := p.StatusEffect("minecraft:poison"); !ok {
		t.Fatal("bee sting did not apply poison")
	}
}

func TestUnprovokedBeeDoesNotSting(t *testing.T) {
	s := newGolemTestServer(t)
	bee := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeBee, 21, 64, 20)
	s.world.Entities.Add(bee)
	p := newBeeTestAggressor(t, s, bee.EntityID)

	ai := s.mobAIFor(bee)
	state := parityState(bee) // angerTicks stays 0: never provoked

	s.tickBeeParity(bee, ai, state)
	if state.beeHasStung {
		t.Fatal("un-angered bee stung a player")
	}
	if p.Health < 20 {
		t.Fatalf("un-angered bee damaged a player: health=%.1f", p.Health)
	}
}

func TestBeeDiesAfterStinging(t *testing.T) {
	s := newGolemTestServer(t)
	bee := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeBee, 21, 64, 20)
	s.world.Entities.Add(bee)

	ai := s.mobAIFor(bee)
	state := parityState(bee)
	state.beeHasStung = true

	// The death chance reaches certainty as the post-sting window closes, so the
	// bee must be dead within the vanilla 1200-tick ceiling.
	for tick := 0; tick < 1300 && !bee.Dead; tick++ {
		s.tickBeeParity(bee, ai, state)
	}
	if !bee.Dead {
		t.Fatal("bee survived past the post-sting death window")
	}
}
