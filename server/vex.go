package server

import (
	corentity "GoCraft/core/entity"
	"GoCraft/java/handler"
)

// tickVexLimitedLife ports Vex.customServerAiStep: a vex summoned with a limited
// life expends it one tick at a time and, once spent, takes 1 starve-style
// damage every 20 ticks until it dies. Vexes without a limited life (should any
// spawn otherwise) are unaffected.
func (s *Server) tickVexLimitedLife(e *corentity.Entity) {
	if e == nil || e.Dead {
		return
	}
	state := parityState(e)
	if !state.hasLimitedLife {
		return
	}
	state.limitedLifeTicks--
	if state.limitedLifeTicks <= 0 {
		state.limitedLifeTicks = 20
		s.damageEnvironmentalEntity(e, 1, "starve")
	}
}

// nearbyVexCount counts living vexes within the given radius of an entity, used
// to cap how many an evoker keeps summoned (vanilla stops at 8).
func (s *Server) nearbyVexCount(origin *corentity.Entity, radius float64) int {
	if s.world == nil {
		return 0
	}
	count := 0
	for _, other := range s.world.Entities.Snapshot() {
		if other != nil && !other.Dead && other.Type == corentity.TypeVex &&
			distance2D(other.Position, origin.Position) <= radius {
			count++
		}
	}
	return count
}

// evokerSummonVexes ports EvokerSummonSpellGoal: it conjures a wave of vexes in a
// small ring above the evoker, each with a randomised limited life (vanilla
// 20*rand(10,39) = 200..780 ticks) so the swarm eventually expires.
func (s *Server) evokerSummonVexes(evoker *corentity.Entity, count int) {
	if s.world == nil || s.game == nil {
		return
	}
	for i := 0; i < count; i++ {
		offsetX := float64((i%3)-1) + 0.5
		offsetZ := float64((i/3)-1) + 0.5
		vex := corentity.New(s.game.NextEntityID(), newRandomUUID(), corentity.TypeVex,
			evoker.Position.X+offsetX, evoker.Position.Y+1, evoker.Position.Z+offsetZ)
		vex.Yaw, vex.Pitch = evoker.Yaw, evoker.Pitch
		state := parityState(vex)
		state.hasLimitedLife = true
		// Deterministic 200..780 spread without a shared RNG dependency.
		state.limitedLifeTicks = 200 + int(uint32(vex.EntityID)*20)%581
		s.world.Entities.Add(vex)
		handler.BroadcastSpawnMob(vex, s.sessions)
	}
}
