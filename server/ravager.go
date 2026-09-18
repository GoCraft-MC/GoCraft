package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/java/handler"
)

// tickRavagerTimers advances the ravager's stun/roar/attack timers every tick
// (Ravager.customServerAiStep). A stun that runs out queues a roar; a roar fires
// its area-of-effect at its midpoint, exactly as vanilla does when roarTick == 10.
func (s *Server) tickRavagerTimers(e *corentity.Entity) {
	if e == nil || e.Dead {
		return
	}
	state := parityState(e)
	if state.ravagerAttackTicks > 0 {
		state.ravagerAttackTicks--
	}
	if state.ravagerRoarTicks > 0 {
		state.ravagerRoarTicks--
		if state.ravagerRoarTicks == 10 {
			s.ravagerRoar(e)
		}
	}
	if state.ravagerStunTicks > 0 {
		state.ravagerStunTicks--
		if state.ravagerStunTicks == 0 {
			state.ravagerRoarTicks = 20
		}
	}
}

// ravagerImmobile reports whether the ravager's own goals are suspended because
// it is stunned, roaring, or in melee recovery (Ravager.isImmobile).
func ravagerImmobile(state *mobParityState) bool {
	return state.ravagerStunTicks > 0 || state.ravagerRoarTicks > 0 || state.ravagerAttackTicks > 0
}

// ravagerRoar deals 6 damage and strong knockback to every living creature within
// 4 blocks, sparing other ravagers and illagers (Ravager.roar).
func (s *Server) ravagerRoar(e *corentity.Entity) {
	for _, sess := range s.allPlayerSessions() {
		p := sess.Player
		if p == nil || p.Dead || p.GameMode == player.GameModeCreative || p.GameMode == player.GameModeSpectator {
			continue
		}
		if ravagerWithinRoar(e, p.Position.X, p.Position.Y, p.Position.Z) {
			handler.DamagePlayerFromSource(sess, ravagerScaledDamage(s, 6), "was struck by a ravager", s.sessions, e.Position.X, e.Position.Z)
			s.sendLegacyPlayerKnockback(sess, e.Position.X, e.Position.Z, 0.9, 0.5)
		}
	}
	if s.world == nil {
		return
	}
	for _, other := range s.world.Entities.Snapshot() {
		if other == nil || other.Dead || other == e || ravagerRoarSpares(other.Type) {
			continue
		}
		if !isPassiveMob(other.Type) && !isHostileMob(other.Type) &&
			other.Type != corentity.TypeIronGolem && other.Type != corentity.TypeSnowGolem {
			continue
		}
		if ravagerWithinRoar(e, other.Position.X, other.Position.Y, other.Position.Z) {
			s.damageEnvironmentalEntity(other, 6, "ravager roar")
		}
	}
}

func ravagerWithinRoar(e *corentity.Entity, x, y, z float64) bool {
	dx, dy, dz := x-e.Position.X, y-e.Position.Y, z-e.Position.Z
	return math.Sqrt(dx*dx+dy*dy+dz*dz) <= 4
}

// ravagerRoarSpares reports whether an entity type is immune to the roar, namely
// other ravagers and the illagers a ravager fights alongside.
func ravagerRoarSpares(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeRavager, corentity.TypePillager, corentity.TypeVindicator,
		corentity.TypeEvoker, corentity.TypeIllusioner:
		return true
	}
	return false
}
