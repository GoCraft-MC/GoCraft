package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestEvokerSummonsShortLivedVexes(t *testing.T) {
	s := newGolemTestServer(t)
	evoker := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeEvoker, 20, 64, 20)
	s.world.Entities.Add(evoker)

	s.evokerSummonVexes(evoker, 3)

	if got := s.nearbyVexCount(evoker, 16); got != 3 {
		t.Fatalf("expected 3 summoned vexes nearby, got %d", got)
	}
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type != corentity.TypeVex {
			continue
		}
		st := parityState(e)
		if !st.hasLimitedLife {
			t.Fatal("summoned vex has no limited life")
		}
		if st.limitedLifeTicks < 200 || st.limitedLifeTicks > 780 {
			t.Fatalf("vex limited life out of vanilla range: %d", st.limitedLifeTicks)
		}
	}
}

func TestLimitedLifeVexEventuallyDies(t *testing.T) {
	s := newGolemTestServer(t)
	vex := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeVex, 20, 64, 20)
	s.world.Entities.Add(vex)
	state := parityState(vex)
	state.hasLimitedLife = true
	state.limitedLifeTicks = 3 // life nearly spent

	// After expiry the vex loses 1 HP every 20 ticks; with <=14 HP it must die
	// within the health*20 + slack window.
	for tick := 0; tick < 400 && !vex.Dead; tick++ {
		s.tickVexLimitedLife(vex)
	}
	if !vex.Dead {
		t.Fatal("limited-life vex never expired")
	}
}

func TestVexWithoutLimitedLifeIsUnaffected(t *testing.T) {
	s := newGolemTestServer(t)
	vex := corentity.New(s.game.NextEntityID(), [16]byte{3}, corentity.TypeVex, 20, 64, 20)
	s.world.Entities.Add(vex)
	startHealth := vex.Health

	for tick := 0; tick < 200; tick++ {
		s.tickVexLimitedLife(vex)
	}
	if vex.Dead || vex.Health != startHealth {
		t.Fatalf("vex without limited life was harmed: dead=%v health=%.1f", vex.Dead, vex.Health)
	}
}
