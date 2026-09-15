package server

import (
	"testing"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func countZombies(s *Server) int {
	n := 0
	for _, e := range s.world.Entities.Snapshot() {
		if e.Type == corentity.TypeZombie {
			n++
		}
	}
	return n
}

func TestHardZombieSummonsReinforcement(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "hard"}
	zombie := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeZombie, 20, 64, 20)
	s.world.Entities.Add(zombie)
	hit := coreworld.EntityDamage{Amount: 2, HasSource: true, SourceX: 10, SourceZ: 20}

	spawned := false
	for i := 0; i < 500 && !spawned; i++ {
		s.tryZombieReinforcement(zombie, hit)
		spawned = countZombies(s) > 1
	}
	if !spawned {
		t.Fatal("hard zombie never summoned a reinforcement over 500 hits")
	}
	// The reinforcement inherits the attacker as a target and cannot summon more.
	for _, e := range s.world.Entities.Snapshot() {
		if e.EntityID == zombie.EntityID || e.Type != corentity.TypeZombie {
			continue
		}
		rai := s.mobAIFor(e)
		if !rai.noReinforce {
			t.Fatal("reinforcement is allowed to summon further reinforcements (chain risk)")
		}
		if !rai.hasTarget {
			t.Fatal("reinforcement did not inherit the attacker as its target")
		}
	}
}

func TestZombieDoesNotReinforceOnNormalDifficulty(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "normal"}
	zombie := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeZombie, 20, 64, 20)
	s.world.Entities.Add(zombie)
	hit := coreworld.EntityDamage{Amount: 2, HasSource: true, SourceX: 10, SourceZ: 20}

	for i := 0; i < 500; i++ {
		s.tryZombieReinforcement(zombie, hit)
	}
	if countZombies(s) != 1 {
		t.Fatalf("zombie count = %d on normal difficulty, want 1 (no reinforcements)", countZombies(s))
	}
}

func TestReinforcementZombieCannotChainReinforce(t *testing.T) {
	s := newGolemTestServer(t)
	s.cfg = &config.Config{Difficulty: "hard"}
	reinf := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeZombie, 20, 64, 20)
	s.world.Entities.Add(reinf)
	s.mobAIFor(reinf).noReinforce = true
	hit := coreworld.EntityDamage{Amount: 2, HasSource: true, SourceX: 10, SourceZ: 20}

	for i := 0; i < 500; i++ {
		s.tryZombieReinforcement(reinf, hit)
	}
	if countZombies(s) != 1 {
		t.Fatalf("reinforcement chained into %d zombies, want 1", countZombies(s))
	}
}
