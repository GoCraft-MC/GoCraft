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
	case corentity.TypeBee:
		return s.tickBeeParity(e, ai, state)
	case corentity.TypeAllay:
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
	case corentity.TypeCat:
		return s.tickCatParity(e, ai, state)
	case corentity.TypeOcelot:
		return s.tickOcelotParity(e, ai, state)
	case corentity.TypeWolf:
		return s.tickTameableFollowParity(e, ai)
	case corentity.TypePolarBear:
		return s.tickPolarBearParity(e, ai, state)
	case corentity.TypeGoat:
		return s.tickGoatParity(e, ai, state)
	case corentity.TypeRabbit:
		return s.tickRabbitParity(e, ai)
	case corentity.TypePanda:
		return s.tickPandaParity(e, ai)
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

// tickBeeParity mirrors vanilla Bee combat: a bee provoked by a player becomes
// angry (refreshParityProvocation sets angerTicks), flies at its aggressor and
// stings once (BeeAttackGoal). Stinging deals 2 damage plus Poison and, exactly
// like vanilla, costs the bee its stinger — it then slowly succumbs, with the
// death chance rising over the ~1200 ticks after stinging. Un-angered bees fall
// back to normal passive flight.
func (s *Server) tickBeeParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if state.beeHasStung {
		state.beeStingTicks++
		if state.beeStingTicks%5 == 0 {
			window := 1200 - state.beeStingTicks
			if window < 1 {
				window = 1
			} else if window > 1200 {
				window = 1200
			}
			if ai.rng.Intn(window) == 0 {
				s.damageEnvironmentalEntity(e, e.MaxHealth, "generic")
				return true
			}
		}
		return s.tickPassiveFlightParity(e, ai, state)
	}
	if state.angerTicks > 0 {
		if aggressor := s.parityAggressorPlayer(e); aggressor != nil {
			return s.tickBeeSting(e, state, aggressor)
		}
	}
	return s.tickPassiveFlightParity(e, ai, state)
}

// tickBeeSting flies the bee toward its aggressor and, once in range with line
// of sight, delivers a single poisonous sting and marks the bee as having stung.
func (s *Server) tickBeeSting(e *corentity.Entity, state *mobParityState, target *player.Player) bool {
	dx := target.Position.X - e.Position.X
	dy := target.Position.Y - e.Position.Y
	dz := target.Position.Z - e.Position.Z
	if math.Sqrt(dx*dx+dy*dy+dz*dz) <= 1.8 {
		if s.mobHasLineOfSight(e, target.Position, 1.62) {
			s.damagePlayerFromParityMob(target, 2, "was stung to death by a bee")
			s.applyParityPlayerPoison(target, 200) // POISON_SECONDS_NORMAL = 10s
			state.beeHasStung = true
			state.beeStingTicks = 0
		}
		e.VX, e.VY, e.VZ = 0, 0, 0
		return true
	}
	s.navigateFlyingMob(e, spatial.Vec3{X: target.Position.X, Y: target.Position.Y + 0.5, Z: target.Position.Z},
		parityFlightSpeed(e.Type, pumpkinMovementSpeed(e.Type, 1.4)))
	return true
}

// parityAggressorPlayer returns the online player in the simulated dimension who
// most recently struck the entity, the vanilla anger target for neutral mobs.
func (s *Server) parityAggressorPlayer(e *corentity.Entity) *player.Player {
	if s.game == nil {
		return nil
	}
	var aggressor *player.Player
	s.game.OnlinePlayers(func(candidate *player.Player) {
		if aggressor == nil && !candidate.Dead && candidate.Dimension == s.simulationDimension &&
			candidate.LastAttackedEntityID == e.EntityID {
			aggressor = candidate
		}
	})
	return aggressor
}

// applyParityPlayerPoison applies (or upgrades) Poison I on a player and syncs it
// to both protocol editions.
func (s *Server) applyParityPlayerPoison(target *player.Player, durationTicks int32) {
	if target == nil {
		return
	}
	stored, changed := target.AddStatusEffect(player.StatusEffect{
		ID: "minecraft:poison", Amplifier: 0, Duration: durationTicks,
		ShowParticles: true, ShowIcon: true,
	})
	if changed {
		s.syncPlayerStatusEffect(target, stored)
	}
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

// tickCatParity mirrors vanilla Cat goals: an untamed cat is skittish and flees
// nearby players (CatAvoidEntityGoal<Player>, 16 blocks, sprint 1.33) and hunts
// rabbits (NonTameRandomTargetGoal + OcelotAttackGoal). A tamed cat follows its
// owner instead.
func (s *Server) tickCatParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if !e.Tamed {
		if target := s.closestVisiblePlayer(e, 16); target != nil {
			dx, dz := e.Position.X-target.Position.X, e.Position.Z-target.Position.Z
			distance := math.Hypot(dx, dz)
			if distance > 0.001 {
				destination := spatial.Vec3{X: e.Position.X + dx/distance*12, Y: e.Position.Y, Z: e.Position.Z + dz/distance*12}
				s.navigateMob(e, ai, destination, pumpkinMovementSpeed(e.Type, 1.33))
				return true
			}
		}
		if prey := s.closestEntityOfTypes(e, 12, corentity.TypeRabbit, corentity.TypeChicken); prey != nil {
			return s.tickEntityHunter(e, ai, state, prey, 3, 20, 1.6, pumpkinMovementSpeed(e.Type, 1.3))
		}
		return false
	}
	return s.tickTameableFollowParity(e, ai)
}

// tickOcelotParity mirrors vanilla Ocelot goals: an untrusting ocelot flees the
// nearest player (OcelotAvoidEntityGoal<Player>, 16 blocks, sprint 1.33), and
// every ocelot hunts chickens and baby turtles on land (OcelotAttackGoal +
// NearestAttackableTargetGoal for Chicken/Turtle). Attack damage is 3.
func (s *Server) tickOcelotParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if !e.Trusting {
		if target := s.closestVisiblePlayer(e, 16); target != nil {
			dx, dz := e.Position.X-target.Position.X, e.Position.Z-target.Position.Z
			distance := math.Hypot(dx, dz)
			if distance > 0.001 {
				destination := spatial.Vec3{X: e.Position.X + dx/distance*12, Y: e.Position.Y, Z: e.Position.Z + dz/distance*12}
				s.navigateMob(e, ai, destination, pumpkinMovementSpeed(e.Type, 1.33))
				return true
			}
		}
	}
	if prey := s.closestEntityMatching(e, 12, func(candidate *corentity.Entity) bool {
		return candidate.Type == corentity.TypeChicken ||
			(candidate.Type == corentity.TypeTurtle && candidate.IsBaby)
	}); prey != nil {
		return s.tickEntityHunter(e, ai, state, prey, 3, 20, 1.6, pumpkinMovementSpeed(e.Type, 1.3))
	}
	return false
}

// tickPandaParity ports the worried panda's storm fright (Panda.isScared): a
// worried panda cowers in place while it is thundering instead of wandering.
func (s *Server) tickPandaParity(e *corentity.Entity, ai *mobAI) bool {
	if e.PandaVariant() == corentity.PandaGeneWorried {
		if _, thundering := s.currentWeather(); thundering {
			e.VX, e.VZ = 0, 0
			clearMobNavigation(e, ai)
			return true
		}
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
	// Rabbits flee nearby threats (players within 8, wolves/foxes within 10) at
	// the vanilla RabbitAvoidEntityGoal 2.2x speed.
	if threat, ok := s.nearestRabbitThreat(e); ok {
		dx, dz := e.Position.X-threat.X, e.Position.Z-threat.Z
		if dist := math.Hypot(dx, dz); dist > 0.001 {
			flee := spatial.Vec3{X: e.Position.X + dx/dist*8, Y: e.Position.Y, Z: e.Position.Z + dz/dist*8}
			ai.hasWanderGoal = false
			s.navigateMob(e, ai, flee, pumpkinMovementSpeed(e.Type, 2.2))
			return true
		}
	}
	if ai.hasWanderGoal {
		return s.navigateMob(e, ai, ai.wanderTarget, pumpkinMovementSpeed(e.Type, 1.0))
	}
	return false
}

// nearestRabbitThreat returns what a rabbit should flee from: the nearest
// non-creative player within 8 blocks or the nearest wolf/fox within 10.
func (s *Server) nearestRabbitThreat(e *corentity.Entity) (spatial.Vec3, bool) {
	var best spatial.Vec3
	found := false
	bestSq := math.MaxFloat64
	if p := s.closestVisiblePlayer(e, 8); p != nil &&
		p.GameMode != player.GameModeCreative && p.GameMode != player.GameModeSpectator {
		best, found, bestSq = p.Position, true, distanceSquaredVec(e.Position, p.Position)
	}
	if pred := s.closestEntityOfTypes(e, 10, corentity.TypeWolf, corentity.TypeFox); pred != nil {
		if d := distanceSquaredVec(e.Position, pred.Position); d < bestSq {
			best, found = pred.Position, true
		}
	}
	return best, found
}

func (s *Server) tickStriderIdleParity(e *corentity.Entity, ai *mobAI) bool {
	if ai.hasWanderGoal {
		return s.navigateStrider(e, ai.wanderTarget, pumpkinMovementSpeed(e.Type, 1.0))
	}
	return false
}
