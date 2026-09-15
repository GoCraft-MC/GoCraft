package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	javaworld "GoCraft/java/world"
)

func TestLightningBoltTypeIsRegistered(t *testing.T) {
	if id := javaworld.EntityTypeID(string(corentity.TypeLightningBolt)); id < 0 {
		t.Fatalf("lightning_bolt entity type unresolved (id=%d); client cannot render the bolt", id)
	}
}

func TestLightningChargesNearbyCreeperOnly(t *testing.T) {
	s := newGolemTestServer(t)
	near := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeCreeper, 22, 64, 20)
	far := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeCreeper, 60, 64, 20)
	s.world.Entities.Add(near)
	s.world.Entities.Add(far)

	s.strikeLightning(20, 64, 20)

	if !near.Charged {
		t.Fatal("creeper within strike radius was not charged")
	}
	if far.Charged {
		t.Fatal("creeper far outside strike radius was charged")
	}
}

func TestLightningBurnsAndDamagesNearbyMob(t *testing.T) {
	s := newGolemTestServer(t)
	zombie := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeZombie, 20.5, 64, 20.5)
	s.world.Entities.Add(zombie)

	s.strikeLightning(20.5, 64, 20.5)

	if zombie.FireTicks < lightningFireTicks {
		t.Fatalf("struck zombie FireTicks = %d, want >= %d", zombie.FireTicks, lightningFireTicks)
	}
	total := float32(0)
	for id, ev := range s.world.DrainEntityDamage() {
		if id == zombie.EntityID {
			total += ev.Amount
		}
	}
	if total < 5 {
		t.Fatalf("struck zombie queued %.1f damage, want >= 5", total)
	}
}

func TestLightningBoltIsSpawned(t *testing.T) {
	s := newGolemTestServer(t)
	before := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeLightningBolt {
			before++
		}
	}
	s.strikeLightning(20, 64, 20)
	after := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeLightningBolt {
			after++
		}
	}
	if after != before+1 {
		t.Fatalf("lightning bolts in world = %d, want %d", after, before+1)
	}
}
