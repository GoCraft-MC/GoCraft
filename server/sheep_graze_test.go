package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func TestShearedSheepRegrowsWoolByEatingGrassBlock(t *testing.T) {
	s := newGolemTestServer(t)
	s.world.SetBlock(20, 63, 20, coreworld.Block{Namespace: "minecraft", Name: "grass_block"})
	s.world.SetBlock(20, 64, 20, coreworld.Air)
	sheep := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSheep, 20.5, 64, 20.5)
	sheep.Sheared = true
	sheep.WoolRegrowTicks = 0 // no timer fallback; only grass-eating can regrow
	s.world.Entities.Add(sheep)

	ate := false
	for i := 0; i < 5000 && !ate; i++ {
		s.tickAnimalLifecycle([]*corentity.Entity{sheep})
		ate = !sheep.Sheared
	}
	if !ate {
		t.Fatal("sheared sheep never regrew wool by eating grass")
	}
	if got := s.world.GetBlock(20, 63, 20).ResourceLocation(); got != "minecraft:dirt" {
		t.Fatalf("eaten grass block = %q, want minecraft:dirt", got)
	}
}

func TestShearedSheepStillRegrowsViaTimerWithoutGrass(t *testing.T) {
	s := newGolemTestServer(t)
	// Stone floor, no grass anywhere.
	s.world.SetBlock(20, 63, 20, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	s.world.SetBlock(20, 64, 20, coreworld.Air)
	sheep := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSheep, 20.5, 64, 20.5)
	sheep.Sheared = true
	sheep.WoolRegrowTicks = 1
	s.world.Entities.Add(sheep)

	s.tickAnimalLifecycle([]*corentity.Entity{sheep})
	if sheep.Sheared {
		t.Fatal("sheep did not regrow wool via the timer fallback")
	}
}
