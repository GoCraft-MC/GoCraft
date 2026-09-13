package server

import (
	"testing"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func placeClosedDoor(s *Server, x, y, z int) {
	s.world.SetBlock(x, y, z, coreworld.Block{Namespace: "minecraft", Name: "oak_door",
		Properties: map[string]string{"half": "lower", "open": "false", "facing": "west"}})
	s.world.SetBlock(x, y+1, z, coreworld.Block{Namespace: "minecraft", Name: "oak_door",
		Properties: map[string]string{"half": "upper", "open": "false", "facing": "west"}})
}

func TestHardZombieBreaksWoodenDoorAfter240Ticks(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "hard"}
	placeClosedDoor(s, 22, 64, 20)
	zombie := corentity.New(s.game.NextEntityID(), [16]byte{9}, corentity.TypeZombie, 21, 64, 20)
	ai := s.mobAIFor(zombie)

	for i := 0; i < zombieDoorBreakTicks-1; i++ {
		if !s.tickZombieDoorBreak(zombie, ai) {
			t.Fatalf("zombie stopped bashing the door at tick %d", i)
		}
	}
	if s.world.GetBlock(22, 64, 20).IsAir() {
		t.Fatal("door broke before 240 ticks")
	}
	s.tickZombieDoorBreak(zombie, ai) // the 240th bash
	if !s.world.GetBlock(22, 64, 20).IsAir() || !s.world.GetBlock(22, 65, 20).IsAir() {
		t.Fatalf("both door halves should be gone: lower=%q upper=%q",
			s.world.GetBlock(22, 64, 20).ResourceLocation(), s.world.GetBlock(22, 65, 20).ResourceLocation())
	}
}

func TestZombieDoesNotBreakDoorOnNormalDifficulty(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "normal"}
	placeClosedDoor(s, 22, 64, 20)
	zombie := corentity.New(s.game.NextEntityID(), [16]byte{9}, corentity.TypeZombie, 21, 64, 20)
	ai := s.mobAIFor(zombie)

	for i := 0; i < zombieDoorBreakTicks+5; i++ {
		if s.tickZombieDoorBreak(zombie, ai) {
			t.Fatal("zombie broke a door on normal difficulty")
		}
	}
	if s.world.GetBlock(22, 64, 20).IsAir() {
		t.Fatal("door was removed on normal difficulty")
	}
}

func TestSkeletonDoesNotBreakDoors(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "hard"}
	placeClosedDoor(s, 22, 64, 20)
	skeleton := corentity.New(s.game.NextEntityID(), [16]byte{9}, corentity.TypeSkeleton, 21, 64, 20)
	ai := s.mobAIFor(skeleton)
	if s.tickZombieDoorBreak(skeleton, ai) {
		t.Fatal("skeleton attempted to break a door")
	}
}
