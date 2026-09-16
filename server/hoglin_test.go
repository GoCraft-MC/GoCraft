package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestHoglinConvertsToZoglinOutsideNether(t *testing.T) {
	s := newGolemTestServer(t) // simulationDimension defaults to the Overworld
	hoglin := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeHoglin, 20, 64, 20)
	s.world.Entities.Add(hoglin)

	for tick := 0; tick <= hoglinConversionTime && !hoglin.Dead; tick++ {
		s.tickHoglinConversion(hoglin)
	}
	if !hoglin.Dead {
		t.Fatalf("hoglin never converted after %d ticks outside the Nether", hoglinConversionTime+1)
	}
	zoglins := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeZoglin {
			zoglins++
		}
	}
	if zoglins != 1 {
		t.Fatalf("expected exactly one zoglin after conversion, got %d", zoglins)
	}
}

func TestHoglinDoesNotConvertInNether(t *testing.T) {
	s := newGolemTestServer(t)
	s.simulationDimension = dimensionNether
	hoglin := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeHoglin, 20, 64, 20)
	s.world.Entities.Add(hoglin)

	for tick := 0; tick < hoglinConversionTime*2; tick++ {
		s.tickHoglinConversion(hoglin)
	}
	if hoglin.Dead {
		t.Fatal("hoglin zombified while in the Nether")
	}
	if got := parityState(hoglin).timeInOverworld; got != 0 {
		t.Fatalf("Nether hoglin accumulated conversion time: %d", got)
	}
}

func TestHoglinConversionTimerResetsOnReturnToNether(t *testing.T) {
	s := newGolemTestServer(t)
	hoglin := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeHoglin, 20, 64, 20)
	s.world.Entities.Add(hoglin)

	for tick := 0; tick < 200; tick++ {
		s.tickHoglinConversion(hoglin)
	}
	if parityState(hoglin).timeInOverworld != 200 {
		t.Fatalf("unexpected accumulated time: %d", parityState(hoglin).timeInOverworld)
	}
	s.simulationDimension = dimensionNether
	s.tickHoglinConversion(hoglin)
	if got := parityState(hoglin).timeInOverworld; got != 0 {
		t.Fatalf("conversion timer did not reset on return to the Nether: %d", got)
	}
}
