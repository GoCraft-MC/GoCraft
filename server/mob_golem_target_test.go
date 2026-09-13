package server

import (
	"math/rand"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func newGolemTestServer(t *testing.T) *Server {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	w.Chunk(0, 0)
	w.Chunk(1, 1)
	return &Server{
		game: game.New(), world: w, sessions: session.NewManager(),
		mobAIs: make(map[int32]*mobAI), spawnRNG: rand.New(rand.NewSource(1)),
	}
}

func TestIronGolemTargetsAnyHostileExceptCreeper(t *testing.T) {
	s := newGolemTestServer(t)
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeIronGolem, 20, 64, 20)
	skeleton := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeSkeleton, 22, 64, 20)
	s.world.Entities.Add(golem)
	s.world.Entities.Add(skeleton)

	s.tickGolemAI(golem)

	ai := s.mobAIFor(golem)
	if !ai.hasTarget || ai.targetEntityID != skeleton.EntityID {
		t.Fatalf("iron golem did not target the skeleton: hasTarget=%v target=%d want=%d", ai.hasTarget, ai.targetEntityID, skeleton.EntityID)
	}
}

func TestIronGolemIgnoresCreeper(t *testing.T) {
	s := newGolemTestServer(t)
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeIronGolem, 20, 64, 20)
	creeper := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeCreeper, 22, 64, 20)
	s.world.Entities.Add(golem)
	s.world.Entities.Add(creeper)

	s.tickGolemAI(golem)

	if ai := s.mobAIFor(golem); ai.hasTarget {
		t.Fatalf("iron golem targeted a creeper (target=%d), which vanilla never does", ai.targetEntityID)
	}
}

func TestSnowGolemThrowsSnowballAtHostile(t *testing.T) {
	s := newGolemTestServer(t)
	golem := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSnowGolem, 20, 64, 20)
	zombie := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeZombie, 24, 64, 20)
	s.world.Entities.Add(golem)
	s.world.Entities.Add(zombie)

	before := countSnowballs(s)
	s.tickGolemAI(golem)
	after := countSnowballs(s)

	if after != before+1 {
		t.Fatalf("snow golem threw %d snowballs, want 1", after-before)
	}
	if ai := s.mobAIFor(golem); ai.attackCooldown != 20 {
		t.Fatalf("snow golem attack cooldown = %d, want 20", ai.attackCooldown)
	}
}

func countSnowballs(s *Server) int {
	count := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeSnowball {
			count++
		}
	}
	return count
}
