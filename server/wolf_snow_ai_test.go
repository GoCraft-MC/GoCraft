package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func TestSnowGolemLeavesSnowTrail(t *testing.T) {
	s := newGolemTestServer(t)
	s.world.SetBlock(20, 63, 20, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	s.world.SetBlock(20, 64, 20, coreworld.Air)
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSnowGolem, 20.5, 64, 20.5)
	s.world.Entities.Add(golem)

	s.tickGolemAI(golem)

	if got := s.world.GetBlock(20, 64, 20).ResourceLocation(); got != "minecraft:snow" {
		t.Fatalf("snow golem left %q under it, want minecraft:snow", got)
	}
}

func TestWolfFleesFromLlama(t *testing.T) {
	s := newGolemTestServer(t)
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeWolf, 20, 64, 20)
	llama := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeLlama, 24, 64, 20)
	s.world.Entities.Add(wolf)
	s.world.Entities.Add(llama)
	ai := s.mobAIFor(wolf)

	if !s.tickWolfBehaviour(wolf, ai) {
		t.Fatal("wolf did not react to the nearby llama")
	}
	if wolf.VX > 0 {
		t.Fatalf("wolf fled toward the llama (llama at +X): vx=%.3f", wolf.VX)
	}
}

func TestWildWolfHuntsPrey(t *testing.T) {
	s := newGolemTestServer(t)
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeWolf, 20, 64, 20)
	sheep := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeSheep, 26, 64, 20)
	s.world.Entities.Add(wolf)
	s.world.Entities.Add(sheep)
	ai := s.mobAIFor(wolf)

	if !s.tickWolfBehaviour(wolf, ai) {
		t.Fatal("wild wolf did not engage nearby prey")
	}
	if ai.targetEntityID != sheep.EntityID {
		t.Fatalf("wild wolf target = %d, want the sheep %d", ai.targetEntityID, sheep.EntityID)
	}
}

func TestTamedWolfDoesNotHuntPrey(t *testing.T) {
	s := newGolemTestServer(t)
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeWolf, 20, 64, 20)
	wolf.Tamed = true
	sheep := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeSheep, 26, 64, 20)
	s.world.Entities.Add(wolf)
	s.world.Entities.Add(sheep)
	ai := s.mobAIFor(wolf)

	// A tamed wolf with no owner cue and no llama should not engage prey.
	if s.tickWolfBehaviour(wolf, ai) {
		t.Fatal("tamed wolf engaged prey like a wild wolf")
	}
}
