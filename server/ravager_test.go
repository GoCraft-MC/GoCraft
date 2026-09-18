package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestRavagerStunExpiresIntoRoar(t *testing.T) {
	s := newGolemTestServer(t)
	rav := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRavager, 20, 64, 20)
	s.world.Entities.Add(rav)
	state := parityState(rav)
	state.ravagerStunTicks = 1

	s.tickRavagerTimers(rav)
	if state.ravagerStunTicks != 0 || state.ravagerRoarTicks != 20 {
		t.Fatalf("stun did not queue a roar: stun=%d roar=%d", state.ravagerStunTicks, state.ravagerRoarTicks)
	}
}

func TestRavagerRoarHitsNearbyPlayer(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{9}, "victim", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Dimension = s.simulationDimension
	p.MaxHealth, p.Health = 20, 20
	p.Position = spatial.Vec3{X: 21, Y: 64, Z: 20}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	rav := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRavager, 20, 64, 20)
	s.world.Entities.Add(rav)
	state := parityState(rav)
	state.ravagerRoarTicks = 11 // next tick lands on the roar midpoint

	s.tickRavagerTimers(rav)
	if p.Health >= 20 {
		t.Fatalf("ravager roar did not damage the nearby player: health=%.1f", p.Health)
	}
}

func TestRavagerRoarSparesIllagersAndDistantMobs(t *testing.T) {
	s := newGolemTestServer(t)
	rav := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRavager, 20, 64, 20)
	pillager := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypePillager, 21, 64, 20)
	cow := corentity.New(s.game.NextEntityID(), [16]byte{3}, corentity.TypeCow, 22, 64, 20)
	far := corentity.New(s.game.NextEntityID(), [16]byte{4}, corentity.TypeCow, 40, 64, 20)
	for _, e := range []*corentity.Entity{rav, pillager, cow, far} {
		s.world.Entities.Add(e)
	}
	pillagerHealth, cowHealth, farHealth := pillager.Health, cow.Health, far.Health

	s.ravagerRoar(rav)

	if pillager.Health != pillagerHealth {
		t.Fatal("ravager roar hit a fellow illager")
	}
	if cow.Health >= cowHealth {
		t.Fatal("ravager roar did not hit a nearby cow")
	}
	if far.Health != farHealth {
		t.Fatal("ravager roar reached a mob beyond 4 blocks")
	}
}
