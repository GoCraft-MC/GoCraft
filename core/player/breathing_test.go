package player

import "testing"

func TestBreathingUsesPumpkinRates(t *testing.T) {
	p := New([16]byte{}, "diver", ClientEditionJava)
	p.GameMode = GameModeSurvival
	for tick := 0; tick < MaxAirSupply; tick++ {
		air, _, drown := p.TickBreathing(true)
		if drown {
			t.Fatalf("drowned early at air %d", air)
		}
	}
	if air := p.AirSupplySnapshot(); air != 0 {
		t.Fatalf("air after 300 ticks = %d, want 0", air)
	}
	for tick := 0; tick < 18; tick++ {
		if _, _, drown := p.TickBreathing(true); drown {
			t.Fatalf("drowned before the 20-tick interval at tick %d", tick)
		}
	}
	if _, _, drown := p.TickBreathing(true); !drown {
		t.Fatal("no drowning damage after the 20-tick interval")
	}
	if air, changed, _ := p.TickBreathing(false); air != 4 || !changed {
		t.Fatalf("first recovery tick = %d/%v, want 4/true", air, changed)
	}
}

func TestCreativeBreathingStaysFull(t *testing.T) {
	p := New([16]byte{}, "builder", ClientEditionBedrock)
	p.AirSupply = 10
	if air, changed, drown := p.TickBreathing(true); air != MaxAirSupply || !changed || drown {
		t.Fatalf("creative breathing = %d/%v/%v", air, changed, drown)
	}
}
