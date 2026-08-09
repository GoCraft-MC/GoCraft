package server

import (
	"math"

	corentity "GoCraft/core/entity"
)

// closestPumpkinEntityTarget implements the non-player ActiveTargetGoal lists
// currently registered by Pumpkin's common mob implementations. Player target
// selection retains its higher priority in tickHostileMobAI.
func (s *Server) closestPumpkinEntityTarget(attacker *corentity.Entity, maximumDistance float64) *corentity.Entity {
	if s.world == nil {
		return nil
	}
	maximumSquared := maximumDistance * maximumDistance
	var closest *corentity.Entity
	for _, candidate := range s.world.Entities.Snapshot() {
		if candidate == attacker || candidate.Dead || !pumpkinMobTargets(attacker.Type, candidate.Type) {
			continue
		}
		dx := candidate.Position.X - attacker.Position.X
		dy := candidate.Position.Y - attacker.Position.Y
		dz := candidate.Position.Z - attacker.Position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance < maximumSquared {
			closest, maximumSquared = candidate, distance
		}
	}
	return closest
}

func pumpkinMobTargets(attacker, candidate corentity.EntityType) bool {
	switch attacker {
	case corentity.TypeZombie, corentity.TypeZombieVillager, corentity.TypeHusk, corentity.TypeDrowned:
		return candidate == corentity.TypeVillager || candidate == corentity.TypeIronGolem || candidate == corentity.TypeTurtle
	case corentity.TypePillager, corentity.TypeVindicator, corentity.TypeEvoker,
		corentity.TypeIllusioner, corentity.TypeRavager:
		return candidate == corentity.TypeVillager || candidate == corentity.TypeIronGolem
	case corentity.TypeGuardian, corentity.TypeElderGuardian:
		return candidate == corentity.TypeSquid || candidate == corentity.TypeGlowSquid || candidate == corentity.TypeAxolotl
	case corentity.TypeEnderman:
		return candidate == corentity.TypeEndermite
	}
	return false
}

func (s *Server) tickHostileAgainstEntity(attacker *corentity.Entity, ai *mobAI, target *corentity.Entity) {
	ai.hasWanderGoal = false
	ai.targetEntityID = target.EntityID
	dx := target.Position.X - attacker.Position.X
	dz := target.Position.Z - attacker.Position.Z
	distance := math.Hypot(dx, dz)
	if distance > 0.001 {
		attacker.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	}
	if ai.attackCooldown > 0 {
		ai.attackCooldown--
	}
	if distance <= 1.8 && math.Abs(target.Position.Y-attacker.Position.Y) <= 2 {
		attacker.VX, attacker.VZ = 0, 0
		if ai.attackCooldown == 0 {
			ai.attackCooldown = 20
			damage := hostileAttackDamage(attacker.Type)
			if settings, ok := pumpkinEntitySpawnSettingsByType[string(attacker.Type)]; ok && settings.attackDamage > 0 {
				damage = float32(settings.attackDamage)
			}
			s.world.QueueEntityDamageFrom(target.EntityID, damage, attacker.Position.X, attacker.Position.Z)
		}
		return
	}
	if isAquaticMob(attacker.Type) && s.entityInWater(attacker) {
		dy := target.Position.Y - attacker.Position.Y
		fullDistance := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if fullDistance > 0.001 {
			speed := pumpkinMovementSpeed(attacker.Type, 1.0)
			attacker.VX, attacker.VY, attacker.VZ = dx/fullDistance*speed, dy/fullDistance*speed, dz/fullDistance*speed
		}
		return
	}
	s.navigateMob(attacker, ai, target.Position, pumpkinMovementSpeed(attacker.Type, 1.0))
}
