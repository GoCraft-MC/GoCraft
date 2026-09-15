package server

import (
	"math"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

// zombieReinforcementChance is the per-hit chance a hurt zombie summons help.
// Vanilla derives this from the SPAWN_REINFORCEMENTS_CHANCE attribute (a random
// 0..0.1 rolled at spawn); GoCraft has no per-entity attribute, so a flat
// hard-difficulty chance stands in.
const zombieReinforcementChance = 0.1

func isReinforceableZombie(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeZombie, corentity.TypeHusk, corentity.TypeZombieVillager, corentity.TypeDrowned:
		return true
	default:
		return false
	}
}

// tryZombieReinforcement mirrors vanilla Zombie reinforcement spawning: on hard
// difficulty a zombie hurt by something has a chance to summon another zombie
// of its kind at a nearby valid spot, which inherits the attacker as its target.
// Summoned reinforcements cannot summon further help, bounding the chain.
func (s *Server) tryZombieReinforcement(e *corentity.Entity, hit coreworld.EntityDamage) {
	if e == nil || s.world == nil || s.game == nil || e.Dead {
		return
	}
	if !isReinforceableZombie(e.Type) || !hit.HasSource || s.currentDifficulty() != 3 {
		return
	}
	ai := s.mobAIFor(e)
	if ai.noReinforce || ai.rng.Float64() >= zombieReinforcementChance {
		return
	}
	spawnX, spawnY, spawnZ, ok := s.findZombieReinforcementSpot(e, ai)
	if !ok {
		return
	}
	reinforcement := corentity.New(s.game.NextEntityID(), newRandomUUID(), e.Type,
		float64(spawnX)+0.5, float64(spawnY), float64(spawnZ)+0.5)
	s.world.Entities.Add(reinforcement)
	handler.BroadcastSpawnMob(reinforcement, s.sessions)

	rai := s.mobAIFor(reinforcement)
	rai.noReinforce = true
	rai.hasTarget = true
	rai.targetEntityID = 0
	rai.targetX, rai.targetZ = hit.SourceX, hit.SourceZ
}

func (s *Server) findZombieReinforcementSpot(e *corentity.Entity, ai *mobAI) (x, y, z int, ok bool) {
	ex := int(math.Floor(e.Position.X))
	ey := int(math.Floor(e.Position.Y))
	ez := int(math.Floor(e.Position.Z))
	for attempt := 0; attempt < 20; attempt++ {
		cx := ex + ai.rng.Intn(15) - 7
		cz := ez + ai.rng.Intn(15) - 7
		groundY := s.world.GroundYAtOrBelow(cx, cz, ey+3)
		if groundY < coreworld.WorldMinY {
			continue
		}
		if cx == ex && cz == ez {
			continue
		}
		return cx, groundY + 1, cz, true
	}
	return 0, 0, 0, false
}
