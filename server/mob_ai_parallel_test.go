package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
)

func TestPassiveAIWorkerCountIsBounded(t *testing.T) {
	tests := []struct {
		entities, processors int
		want                 int
	}{
		{entities: 0, processors: 16, want: 0},
		{entities: 1, processors: 16, want: 1},
		{entities: 20, processors: 1, want: 1},
		{entities: 20, processors: 4, want: 4},
		{entities: 20, processors: 64, want: maximumPassiveAIWorkers},
		{entities: 3, processors: 4, want: 3},
	}
	for _, test := range tests {
		if got := passiveAIWorkerCount(test.entities, test.processors); got != test.want {
			t.Errorf("passiveAIWorkerCount(%d, %d) = %d, want %d", test.entities, test.processors, got, test.want)
		}
	}
}

func TestPassiveAIParallelProcessesEveryCandidateExactlyOnce(t *testing.T) {
	s := &Server{mobAIs: make(map[int32]*mobAI)}
	const candidateCount = 64
	entities := make([]*corentity.Entity, 0, candidateCount+2)
	for id := int32(1); id <= candidateCount; id++ {
		entity := corentity.New(id, [16]byte{}, corentity.TypeCow, 1.5, 64, 1.5)
		entities = append(entities, entity)
		ai := s.mobAIFor(entity)
		ai.knockbackTick = 2
	}
	far := corentity.New(1000, [16]byte{}, corentity.TypeCow, 500, 64, 500)
	farAI := s.mobAIFor(far)
	farAI.knockbackTick = 2
	entities = append(entities, far)
	dead := corentity.New(1001, [16]byte{}, corentity.TypeCow, 1.5, 64, 1.5)
	dead.Dead = true
	deadAI := s.mobAIFor(dead)
	deadAI.knockbackTick = 2
	entities = append(entities, dead)

	players := []naturalSpawnPlayer{{id: 2000, position: spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}}}
	if wakes := s.tickPassiveAIParallel(entities, players); len(wakes) != 0 {
		t.Fatalf("ordinary cows produced villager wake events: %d", len(wakes))
	}
	for id := int32(1); id <= candidateCount; id++ {
		if got := s.mobAIs[id].knockbackTick; got != 1 {
			t.Errorf("candidate %d knockback tick = %d, want 1", id, got)
		}
	}
	if farAI.knockbackTick != 2 || deadAI.knockbackTick != 2 {
		t.Fatalf("excluded AI was processed: far=%d dead=%d", farAI.knockbackTick, deadAI.knockbackTick)
	}
}
