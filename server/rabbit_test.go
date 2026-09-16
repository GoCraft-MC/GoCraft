package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestRabbitFleesFromNearbyWolf(t *testing.T) {
	s := newGolemTestServer(t)
	rabbit := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRabbit, 20, 64, 20)
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeWolf, 23, 64, 20)
	s.world.Entities.Add(rabbit)
	s.world.Entities.Add(wolf)
	ai := s.mobAIFor(rabbit)

	if !s.tickRabbitParity(rabbit, ai) {
		t.Fatal("rabbit did not flee the nearby wolf")
	}
	if rabbit.VX > 0 {
		t.Fatalf("rabbit fled toward the wolf (wolf at +X): vx=%.3f", rabbit.VX)
	}
}

func TestRabbitFleesFromNearbyFox(t *testing.T) {
	s := newGolemTestServer(t)
	rabbit := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRabbit, 20, 64, 20)
	fox := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeFox, 20, 64, 25)
	s.world.Entities.Add(rabbit)
	s.world.Entities.Add(fox)
	ai := s.mobAIFor(rabbit)

	if _, ok := s.nearestRabbitThreat(rabbit); !ok {
		t.Fatal("fox not recognised as a rabbit threat")
	}
	if !s.tickRabbitParity(rabbit, ai) {
		t.Fatal("rabbit did not flee the nearby fox")
	}
	if rabbit.VZ > 0 {
		t.Fatalf("rabbit fled toward the fox (fox at +Z): vz=%.3f", rabbit.VZ)
	}
}

func TestRabbitIgnoresDistantPredator(t *testing.T) {
	s := newGolemTestServer(t)
	rabbit := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRabbit, 20, 64, 20)
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeWolf, 40, 64, 20) // >10 blocks
	s.world.Entities.Add(rabbit)
	s.world.Entities.Add(wolf)

	if _, ok := s.nearestRabbitThreat(rabbit); ok {
		t.Fatal("rabbit treated a >10-block wolf as a threat")
	}
}
