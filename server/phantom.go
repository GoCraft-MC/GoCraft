package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

// tickPhantomSwoop ports the Phantom attack strategy (AttackPhase CIRCLE/SWOOP):
// instead of pursuing at head height like a ground mob, a phantom orbits high
// above its target and periodically dives to strike, then climbs back to circle.
func (s *Server) tickPhantomSwoop(e *corentity.Entity, ai *mobAI, target *session.Session, distance float64, visible bool) {
	state := parityState(e)
	tp := target.Player.Position
	if ai.attackCooldown > 0 {
		ai.attackCooldown--
	}

	if state.phantomSwooping {
		// Contact ends the dive with a hit; the phantom then peels away to circle,
		// mirroring SwoopAttackGoal stopping on doHurtTarget.
		if distance <= 2.5 && math.Abs(e.Position.Y-tp.Y) <= 3 && visible {
			if ai.attackCooldown == 0 {
				ai.attackCooldown = 20
				s.damagePhantomSwoopTarget(e, target)
			}
			state.phantomSwooping = false
			state.phantomCircleTicks = 60 + ai.rng.Intn(60)
			return
		}
		// Overshot above the target without connecting: give up and circle again.
		if e.Position.Y > tp.Y+8 {
			state.phantomSwooping = false
			state.phantomCircleTicks = 40 + ai.rng.Intn(40)
		}
		s.navigateFlyingMob(e, tp, pumpkinMovementSpeed(e.Type, 1.8))
		return
	}

	// CIRCLE: orbit ~7 blocks above the target until the next sweep is due.
	if state.phantomCircleTicks > 0 {
		state.phantomCircleTicks--
	}
	angle := float64(e.AgeTicks) * 0.15
	orbit := spatial.Vec3{X: tp.X + math.Cos(angle)*6, Y: tp.Y + 7, Z: tp.Z + math.Sin(angle)*6}
	s.navigateFlyingMob(e, orbit, pumpkinMovementSpeed(e.Type, 1.0))
	if state.phantomCircleTicks == 0 && distance < 32 {
		state.phantomSwooping = true
	}
}

// damagePhantomSwoopTarget applies a swoop hit with the same difficulty scaling
// and knockback the ground-melee controller uses.
func (s *Server) damagePhantomSwoopTarget(e *corentity.Entity, target *session.Session) {
	damage := float32(6) // vanilla ATTACK_DAMAGE base for a size-0 phantom
	switch s.currentDifficulty() {
	case 1:
		damage *= 0.5
	case 3:
		damage *= 1.5
	}
	healthBefore, _, _, _ := target.Player.HealthSnapshot()
	handler.DamagePlayerFromSource(target, damage, "was slain by a phantom", s.sessions, e.Position.X, e.Position.Z)
	healthAfter, _, _, _ := target.Player.HealthSnapshot()
	if healthAfter < healthBefore {
		target.Player.LastAttackerEntityID = e.EntityID
		s.sendLegacyPlayerKnockback(target, e.Position.X, e.Position.Z, 0.4, 0.4)
	}
}
