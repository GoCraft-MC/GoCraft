package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

// tickRavagerCombat drives ravager pursuit and melee. A hit that a player blocks
// with a shield may stun the ravager (Ravager.blockedByItem: 50% chance),
// otherwise it shoves the blocker back.
func (s *Server) tickRavagerCombat(e *corentity.Entity, ai *mobAI, target *session.Session, distance float64, visible bool) {
	state := parityState(e)
	if ravagerImmobile(state) {
		e.VX, e.VZ = 0, 0
		clearMobNavigation(e, ai)
		return
	}
	if ai.attackCooldown > 0 {
		ai.attackCooldown--
	}
	if distance <= 3.0 && visible && ai.attackCooldown == 0 {
		state.ravagerAttackTicks = 10
		ai.attackCooldown = 20
		e.VX, e.VZ = 0, 0
		healthBefore, _, _, _ := target.Player.HealthSnapshot()
		landed := handler.DamagePlayerFromSource(target, ravagerScaledDamage(s, 12), "was struck by a ravager", s.sessions, e.Position.X, e.Position.Z)
		if landed {
			if after, _, _, _ := target.Player.HealthSnapshot(); after < healthBefore {
				target.Player.LastAttackerEntityID = e.EntityID
				s.sendLegacyPlayerKnockback(target, e.Position.X, e.Position.Z, 0.4, 0.4)
			}
			return
		}
		// A shield facing the ravager ate the blow: 50% stun, else shove the blocker.
		if playerBlockingHitFrom(target.Player, e.Position.X, e.Position.Z) {
			if state.ravagerRoarTicks == 0 && ai.rng.Float64() < 0.5 {
				state.ravagerStunTicks = 40
			} else {
				s.sendLegacyPlayerKnockback(target, e.Position.X, e.Position.Z, 0.5, 0.5)
			}
		}
		return
	}
	if distance > 0.001 {
		s.navigateMob(e, ai, target.Player.Position, pumpkinMovementSpeed(e.Type, 1.0))
	}
}

func ravagerScaledDamage(s *Server, base float32) float32 {
	switch s.currentDifficulty() {
	case 1:
		return base * 0.5
	case 3:
		return base * 1.5
	}
	return base
}

// playerBlockingHitFrom reports whether a player is actively blocking with a
// raised shield facing an attack coming from (srcX, srcZ), mirroring the shield
// check in the damage pipeline.
func playerBlockingHitFrom(p *player.Player, srcX, srcZ float64) bool {
	if p == nil {
		return false
	}
	shieldInMain := p.Inventory[player.HotbarStart+p.HeldSlot].ItemID == "minecraft:shield"
	shieldInOff := p.Inventory[player.OffhandSlot].ItemID == "minecraft:shield"
	if !(shieldInMain || shieldInOff) || p.UsingItemID != "minecraft:shield" {
		return false
	}
	yawRad := float64(p.Rotation.Yaw) * math.Pi / 180
	lookX, lookZ := -math.Sin(yawRad), math.Cos(yawRad)
	dx, dz := p.Position.X-srcX, p.Position.Z-srcZ
	if dist := math.Hypot(dx, dz); dist > 0.001 {
		dx, dz = dx/dist, dz/dist
	}
	return lookX*dx+lookZ*dz < 0
}
