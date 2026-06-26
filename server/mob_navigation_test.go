package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestNavigatorRepathsAreStaggeredAndRespectCooldown(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	s := &Server{world: world}
	first := corentity.New(1, [16]byte{}, corentity.TypeCow, 1.5, 64, 1.5)
	second := corentity.New(2, [16]byte{}, corentity.TypeCow, 1.5, 64, 2.5)
	firstAI, secondAI := &mobAI{}, &mobAI{}
	firstGoal := spatial.Vec3{X: 6.5, Y: 64, Z: 1.5}
	if !s.navigateMob(first, firstAI, firstGoal, 0.1) || !s.navigateMob(second, secondAI, firstGoal, 0.1) {
		t.Fatal("initial navigation did not produce paths")
	}
	if firstAI.repathTick == secondAI.repathTick {
		t.Fatalf("entity repath phases were aligned at %d ticks", firstAI.repathTick)
	}
	originalGoal := firstAI.pathGoal
	s.navigateMob(first, firstAI, spatial.Vec3{X: 10.5, Y: 64, Z: 1.5}, 0.1)
	if firstAI.pathGoal != originalGoal {
		t.Fatalf("moving target forced A* during cooldown: old=%+v new=%+v", originalGoal, firstAI.pathGoal)
	}
}
