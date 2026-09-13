package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

// tickParityPassiveIdle owns idle behaviour for passive/neutral mobs whose
// vanilla goals are more specific than generic ground wandering. Higher
// priorities (panic, breeding, riding and villager Brain) run before this hook.
func (s *Server) tickParityPassiveIdle(e *corentity.Entity, ai *mobAI) bool {
	if e == nil || ai == nil {
		return false
	}
	state := parityState(e)
	tickParityCooldowns(state)

	switch e.Type {
	case corentity.TypeBat:
		return s.tickBatParity(e, ai, state)
	case corentity.TypeAllay, corentity.TypeBee:
		return s.tickPassiveFlightParity(e, ai, state)
	case corentity.TypeParrot:
		if s.tickTameableFollowParity(e, ai) {
			return true
		}
		return s.tickPassiveFlightParity(e, ai, state)
	case corentity.TypeCod, corentity.TypeSalmon, corentity.TypeTropicalFish:
		return s.tickSchoolingFishParity(e, ai)
	case corentity.TypeDolphin:
		return s.tickDolphinParity(e, ai)
	case corentity.TypeAxolotl:
		return s.tickAxolotlParity(e, ai, state)
	case corentity.TypeFrog:
		return s.tickFrogParity(e, ai, state)
	case corentity.TypeFox:
		return s.tickFoxParity(e, ai, state)
	case corentity.TypeCat, corentity.TypeWolf:
		return s.tickTameableFollowParity(e, ai)
	case corentity.TypePolarBear:
		return s.tickPolarBearParity(e, ai, state)
	case corentity.TypeGoat:
		return s.tickGoatParity(e, ai, state)
	case corentity.TypeRabbit:
		return s.tickRabbitParity(e, ai)
	case corentity.TypeStrider:
		return s.tickStriderIdleParity(e, ai)
	case corentity.TypeSquid, corentity.TypeGlowSquid, corentity.TypePufferfish, corentity.TypeTadpole:
		if s.entityInWater(e) {
			s.tickAquaticMobAI(e, ai)
			return true
		}
	}
	return false
}

func (s *Server) tickPassiveFlightParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if e.Tamed && e.HasTameOwner && s.tickTameableFollowParity(e, ai) {
		return true
	}
	if !state.hasAnchor || state.phaseTicks <= 0 || distanceSquaredVec(e.Position, state.anchor) < 1 {
		state.anchor = spatial.Vec3{
			X: e.Position.X + ai.rng.Float64()*16 - 8,
			Y: e.Position.Y + ai.rng.Float64()*8 - 4,
			Z: e.Position.Z + ai.rng.Float64()*16 - 8,
		}
		state.hasAnchor = true
		state.phaseTicks = 30 + ai.rng.Intn(50)
	}
	s.navigateFlyingMob(e, state.anchor, parityFlightSpeed(e.Type, pumpkinMovementSpeed(e.Type, 1.0)))
	return true
}

func (s *Server) tickBatParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if s.world == nil {
		return false
	}
	x := int(math.Floor(e.Position.X))
	y := int(math.Floor(e.Position.Y))
	z := int(math.Floor(e.Position.Z))
	above := s.world.GetBlock(x, y+1, z).ResourceLocation()
	dayTime := ((s.worldAge % 24000) + 24000) % 24000
	if dayTime < 12000 && above != "minecraft:air" && above != "minecraft:water" && above != "minecraft:lava" {
		e.VX, e.VY, e.VZ = 0, 0, 0
		state.hasAnchor = false
		return true
	}
	return s.tickPassiveFlightParity(e, ai, state)
}

func (s *Server) tickSchoolingFishParity(e *corentity.Entity, ai *mobAI) bool {
	if !s.entityInWater(e) || s.world == nil {
		return false
	}
	if leader := s.closestEntityOfTypes(e, 12, e.Type); leader != nil {
		s.navigateSwimmingMob(e, leader.Position, pumpkinMovementSpeed(e.Type, 1.0))
		return true
	}
	s.tickAquaticMobAI(e, ai)
	return true
}

func (s *Server) tickDolphinParity(e *corentity.Entity, ai *mobAI) bool {
	if !s.entityInWater(e) {
		return false
	}
	if target := s.closestVisiblePlayer(e, 10); target != nil {
		s.navigateSwimmingMob(e, spatial.Vec3{X: target.Position.X, Y: target.Position.Y + 0.5, Z: target.Position.Z}, pumpkinMovementSpeed(e.Type, 1.2))
		return true
	}
	s.tickAquaticMobAI(e, ai)
	return true
}

func (s *Server) tickAxolotlParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if !s.entityInWater(e) {
		return false
	}
	if prey := s.closestEntityOfTypes(e, 8,
		corentity.TypeCod, corentity.TypeSalmon, corentity.TypeTropicalFish,
		corentity.TypeSquid, corentity.TypeGlowSquid, corentity.TypeTadpole); prey != nil {
		return s.tickEntityHunter(e, ai, state, prey, 2, 20, 1.5, pumpkinMovementSpeed(e.Type, 1.2))
	}
	s.tickAquaticMobAI(e, ai)
	return true
}

func (s *Server) tickFrogParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if prey := s.closestEntityOfTypes(e, 10, corentity.TypeSlime, corentity.TypeMagmaCube); prey != nil {
		return s.tickEntityHunter(e, ai, state, prey, 2, 20, 1.6, pumpkinMovementSpeed(e.Type, 1.2))
	}
	if s.entityInWater(e) {
		s.tickAquaticMobAI(e, ai)
		return true
	}
	return false
}

func (s *Server) tickFoxParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if !e.Trusting {
		if target := s.closestVisiblePlayer(e, 16); target != nil {
			dx, dz := e.Position.X-target.Position.X, e.Position.Z-target.Position.Z
			distance := math.Hypot(dx, dz)
			if distance > 0.001 {
				destination := spatial.Vec3{X: e.Position.X + dx/distance*12, Y: e.Position.Y, Z: e.Position.Z + dz/distance*12}
				s.navigateMob(e, ai, destination, pumpkinMovementSpeed(e.Type, 1.6))
				return true
			}
		}
	}
	if prey := s.closestEntityOfTypes(e, 12, corentity.TypeChicken, corentity.TypeRabbit, corentity.TypeCod, corentity.TypeSalmon); prey != nil {
		return s.tickEntityHunter(e, ai, state, prey, 2, 20, 1.6, pumpkinMovementSpeed(e.Type, 1.3))
	}
	return false
}

func (s *Server) tickTameableFollowParity(e *corentity.Entity, ai *mobAI) bool {
	if !e.Tamed || !e.HasTameOwner || e.Sitting || s.game == nil {
		return false
	}
	var owner *player.Player
	s.game.OnlinePlayers(func(candidate *player.Player) {
		if owner == nil && candidate.UUID == e.TameOwnerUUID && candidate.Dimension == s.simulationDimension && !candidate.Dead {
			owner = candidate
		}
	})
	if owner == nil {
		return false
	}
	dx, dz := owner.Position.X-e.Position.X, owner.Position.Z-e.Position.Z
	distanceSquared := dx*dx + dz*dz
	if distanceSquared > 12*12 {
		// FollowOwnerGoal teleports tameables that cannot keep up. Keep the
		// destination next to, rather than inside, the owner.
		e.Position = spatial.Vec3{X: owner.Position.X + 1, Y: owner.Position.Y, Z: owner.Position.Z + 1}
		e.VX, e.VY, e.VZ = 0, 0, 0
		clearMobNavigation(e, ai)
		return true
	}
	if distanceSquared > 4*4 {
		s.navigateMob(e, ai, owner.Position, pumpkinMovementSpeed(e.Type, 1.2))
		return true
	}
	return false
}

func (s *Server) tickPolarBearParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if e.IsBaby || s.game == nil {
		return false
	}
	cubNearby := s.closestEntityMatching(e, 12, func(candidate *corentity.Entity) bool {
		return candidate.Type == corentity.TypePolarBear && candidate.IsBaby
	}) != nil
	if !cubNearby {
		return false
	}
	target := s.closestVisiblePlayer(e, 16)
	if target == nil {
		return false
	}
	return s.tickPlayerHunter(e, ai, state, target, 6, 25, 2.0, pumpkinMovementSpeed(e.Type, 1.25))
}

func (s *Server) tickGoatParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if e.IsBaby || state.primaryCooldown > 0 {
		return false
	}
	target := s.closestVisiblePlayer(e, 10)
	if target == nil {
		return false
	}
	dx, dz := target.Position.X-e.Position.X, target.Position.Z-e.Position.Z
	distance := math.Hypot(dx, dz)
	if distance < 0.001 {
		return false
	}
	if distance <= 2.0 {
		if s.mobHasLineOfSight(e, target.Position, 1.62) {
			s.damagePlayerFromParityMob(target, 2, "was rammed by a goat")
			s.sendParityPlayerVelocity(target, dx/distance*1.5, 0.35, dz/distance*1.5)
		}
		state.primaryCooldown = 600
		e.VX, e.VZ = 0, 0
		return true
	}
	s.navigateMob(e, ai, target.Position, pumpkinMovementSpeed(e.Type, 1.8))
	return true
}

func (s *Server) tickRabbitParity(e *corentity.Entity, ai *mobAI) bool {
	if ai.hasWanderGoal {
		return s.navigateMob(e, ai, ai.wanderTarget, pumpkinMovementSpeed(e.Type, 1.0))
	}
	return false
}

func (s *Server) tickStriderIdleParity(e *corentity.Entity, ai *mobAI) bool {
	if ai.hasWanderGoal {
		return s.navigateStrider(e, ai.wanderTarget, pumpkinMovementSpeed(e.Type, 1.0))
	}
	return false
}
