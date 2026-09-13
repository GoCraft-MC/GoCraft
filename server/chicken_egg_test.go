package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestChickenLaysEggWhenTimerElapses(t *testing.T) {
	s := newGolemTestServer(t)
	chicken := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeChicken, 20, 64, 20)
	chicken.EggLayTicks = 1
	s.world.Entities.Add(chicken)

	s.tickAnimalLifecycle([]*corentity.Entity{chicken})

	eggs := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeItem && e.DroppedItem().ItemID == "minecraft:egg" {
			eggs++
		}
	}
	if eggs != 1 {
		t.Fatalf("chicken dropped %d eggs, want 1", eggs)
	}
	if chicken.EggLayTicks < 6000 || chicken.EggLayTicks > 12000 {
		t.Fatalf("egg timer re-roll = %d, want 6000-12000", chicken.EggLayTicks)
	}
}

func TestBabyChickenDoesNotLayEgg(t *testing.T) {
	s := newGolemTestServer(t)
	chick := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeChicken, 20, 64, 20)
	chick.IsBaby = true
	chick.EggLayTicks = 1
	s.world.Entities.Add(chick)

	s.tickAnimalLifecycle([]*corentity.Entity{chick})

	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeItem && e.DroppedItem().ItemID == "minecraft:egg" {
			t.Fatal("baby chicken laid an egg")
		}
	}
}

func TestAdultChickenInitialisesEggTimerWithoutImmediateLay(t *testing.T) {
	s := newGolemTestServer(t)
	chicken := corentity.New(s.game.NextEntityID(), [16]byte{3}, corentity.TypeChicken, 20, 64, 20)
	chicken.EggLayTicks = 0 // uninitialised
	s.world.Entities.Add(chicken)

	s.tickAnimalLifecycle([]*corentity.Entity{chicken})

	if chicken.EggLayTicks < 5999 || chicken.EggLayTicks > 12000 {
		t.Fatalf("egg timer init = %d, want ~6000-12000", chicken.EggLayTicks)
	}
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeItem {
			t.Fatal("chicken laid an egg on the first tick after spawning")
		}
	}
}
