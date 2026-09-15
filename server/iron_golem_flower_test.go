package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestIronGolemOffersFlowerToNearbyVillagerByDay(t *testing.T) {
	s := newGolemTestServer(t)
	s.worldAge = 1000 // daytime
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeIronGolem, 20, 64, 20)
	villager := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeVillager, 22, 64, 20)
	s.world.Entities.Add(golem)
	s.world.Entities.Add(villager)
	ai := s.mobAIFor(golem)

	offered := false
	for i := 0; i < 200000 && !offered; i++ {
		s.tickIronGolemOfferFlower(golem, ai)
		offered = golem.OfferFlowerTicks > 0
	}
	if !offered {
		t.Fatal("iron golem never offered a flower to the nearby villager")
	}
	if golem.OfferFlowerTicks != 400 {
		t.Fatalf("offer duration = %d, want 400", golem.OfferFlowerTicks)
	}
}

func TestIronGolemDoesNotOfferFlowerAtNight(t *testing.T) {
	s := newGolemTestServer(t)
	s.worldAge = 14000 // night
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeIronGolem, 20, 64, 20)
	villager := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeVillager, 22, 64, 20)
	s.world.Entities.Add(golem)
	s.world.Entities.Add(villager)
	ai := s.mobAIFor(golem)

	for i := 0; i < 50000; i++ {
		s.tickIronGolemOfferFlower(golem, ai)
	}
	if golem.OfferFlowerTicks > 0 {
		t.Fatal("iron golem offered a flower at night")
	}
}

func TestIronGolemDoesNotOfferFlowerWithoutVillager(t *testing.T) {
	s := newGolemTestServer(t)
	s.worldAge = 1000
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeIronGolem, 20, 64, 20)
	s.world.Entities.Add(golem)
	ai := s.mobAIFor(golem)

	for i := 0; i < 50000; i++ {
		s.tickIronGolemOfferFlower(golem, ai)
	}
	if golem.OfferFlowerTicks > 0 {
		t.Fatal("iron golem offered a flower with no villager nearby")
	}
}
