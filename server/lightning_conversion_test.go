package server

import (
	"testing"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
)

func entityGone(s *Server, id int32) bool {
	for _, e := range s.world.Entities.Snapshot() {
		if e.EntityID == id {
			return false
		}
	}
	return true
}

func countType(s *Server, tp corentity.EntityType) int {
	n := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == tp {
			n++
		}
	}
	return n
}

func TestLightningConvertsPigToZombifiedPiglin(t *testing.T) {
	s := newGolemTestServer(t) // default difficulty is normal
	pig := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePig, 20.5, 64, 20.5)
	s.world.Entities.Add(pig)

	s.strikeLightning(20.5, 64, 20.5)

	if !entityGone(s, pig.EntityID) {
		t.Fatal("pig was not removed after conversion")
	}
	found := false
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeZombifiedPiglin && e.MainHandItemID == "minecraft:golden_sword" {
			found = true
		}
	}
	if !found {
		t.Fatal("no zombified piglin with a golden sword spawned")
	}
}

func TestLightningConvertsVillagerToWitch(t *testing.T) {
	s := newGolemTestServer(t)
	villager := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeVillager, 20.5, 64, 20.5)
	s.world.Entities.Add(villager)

	s.strikeLightning(20.5, 64, 20.5)

	if !entityGone(s, villager.EntityID) {
		t.Fatal("villager was not removed after conversion")
	}
	if countType(s, corentity.TypeWitch) != 1 {
		t.Fatalf("witches after strike = %d, want 1", countType(s, corentity.TypeWitch))
	}
}

func TestLightningDoesNotConvertOnPeaceful(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "peaceful"}
	pig := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypePig, 20.5, 64, 20.5)
	s.world.Entities.Add(pig)

	s.strikeLightning(20.5, 64, 20.5)

	if entityGone(s, pig.EntityID) {
		t.Fatal("pig was converted on peaceful difficulty")
	}
}

func TestLightningTogglesMooshroomVariant(t *testing.T) {
	s := newGolemTestServer(t)
	moo := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeMooshroom, 20.5, 64, 20.5)
	s.world.Entities.Add(moo)

	s.strikeLightning(20.5, 64, 20.5)
	if !moo.MooshroomBrown {
		t.Fatal("red mooshroom did not toggle to brown")
	}
}
