package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	"GoCraft/java/handler"
)

const (
	wolfMeleeReach = 1.8
	wolfChaseSpeed = 0.3
)

// wolfPrey mirrors the wild wolf targeting set: Wolf.PREY_SELECTOR (sheep,
// rabbit, fox) plus NearestAttackableTargetGoal(AbstractSkeleton).
func wolfPrey(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeSheep, corentity.TypeRabbit, corentity.TypeFox,
		corentity.TypeSkeleton, corentity.TypeStray, corentity.TypeWitherSkeleton, corentity.TypeBogged:
		return true
	default:
		return false
	}
}

// tickWolfBehaviour runs the wolf goals not owned by owner-assist combat:
// AvoidEntityGoal(Llama, 24) and, for untamed wolves, hunting prey. Returns
// true when it owns the tick so the shared passive wander is skipped.
func (s *Server) tickWolfBehaviour(e *corentity.Entity, ai *mobAI) bool {
	if e == nil || ai == nil || s.world == nil || e.Sitting {
		return false
	}
	if s.tickWolfAvoidLlama(e, ai) {
		return true
	}
	if e.Tamed {
		return s.tickTamedWolfCombat(e, ai)
	}
	return s.tickWildWolfCombat(e, ai)
}

func (s *Server) tickWolfAvoidLlama(e *corentity.Entity, ai *mobAI) bool {
	llama := s.closestEntityOfTypes(e, 24, corentity.TypeLlama, corentity.TypeTraderLlama)
	if llama == nil {
		return false
	}
	dx, dz := e.Position.X-llama.Position.X, e.Position.Z-llama.Position.Z
	distance := math.Hypot(dx, dz)
	if distance < 0.001 {
		return false
	}
	flee := spatial.Vec3{X: e.Position.X + dx/distance*10, Y: e.Position.Y, Z: e.Position.Z + dz/distance*10}
	s.navigateMob(e, ai, flee, pumpkinMovementSpeed(e.Type, 1.5))
	return true
}

func (s *Server) tickWildWolfCombat(e *corentity.Entity, ai *mobAI) bool {
	var target *corentity.Entity
	if ai.hasTarget && ai.targetEntityID != 0 {
		if existing, ok := s.world.Entities.Get(ai.targetEntityID); ok && !existing.Dead && wolfPrey(existing.Type) {
			target = existing
		}
	}
	if target == nil {
		target = s.closestEntityMatching(e, 16, func(c *corentity.Entity) bool { return wolfPrey(c.Type) })
	}
	if target == nil {
		ai.hasTarget = false
		ai.targetEntityID = 0
		return false
	}
	ai.hasTarget = true
	ai.targetEntityID = target.EntityID
	ai.hasWanderGoal = false

	dx, dz := target.Position.X-e.Position.X, target.Position.Z-e.Position.Z
	dist := math.Hypot(dx, dz)
	if dist > 0.001 {
		e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	}
	if ai.attackCooldown > 0 {
		ai.attackCooldown--
	}
	if dist <= wolfMeleeReach && ai.attackCooldown == 0 {
		ai.attackCooldown = 20
		if s.mobHasLineOfSight(e, target.Position, 1.4) {
			s.world.QueueEntityDamageFrom(target.EntityID, 4, e.Position.X, e.Position.Z)
			handler.BroadcastSoundAt(s.sessions, "minecraft:entity.wolf.hurt", handler.SoundCategoryHostile,
				e.Position.X, e.Position.Y, e.Position.Z, 1, 1)
		}
		e.VX, e.VZ = 0, 0
		return true
	}
	s.wolfMaybeLeap(e, ai, dx, dz, dist)
	if !s.navigateMob(e, ai, spatial.Vec3{X: target.Position.X, Y: e.Position.Y, Z: target.Position.Z}, wolfChaseSpeed) && dist > 0.001 {
		e.VX, e.VZ = dx/dist*wolfChaseSpeed, dz/dist*wolfChaseSpeed
	}
	return true
}

// wolfMaybeLeap mirrors LeapAtTargetGoal(0.4): a grounded wolf 2-4 blocks from
// its target occasionally pounces, adding upward and forward velocity.
func (s *Server) wolfMaybeLeap(e *corentity.Entity, ai *mobAI, dx, dz, dist float64) {
	if !e.OnGround || dist < 2 || dist > 4 || dist < 0.001 {
		return
	}
	if ai.rng.Float64() >= 0.2 {
		return
	}
	e.VY = 0.4
	e.VX += dx / dist * 0.4
	e.VZ += dz / dist * 0.4
}
