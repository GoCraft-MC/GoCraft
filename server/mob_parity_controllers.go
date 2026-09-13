package server

import (
	"math"
	"sync"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

// mobParityState contains state used by mob-specific controllers that does not
// belong in the generic mobAI pathfinder state. The canonical entity remains
// the source of truth for persistent state; these values mirror vanilla's
// transient Goal/Brain cooldowns and activity timers.
type mobParityState struct {
	primaryCooldown   int
	secondaryCooldown int
	angerTicks        int
	jumpCooldown      int
	phaseTicks         int
	anchor             spatial.Vec3
	hasAnchor          bool
	targetEntityID     int32
}

var mobParityStates sync.Map // map[*corentity.Entity]*mobParityState

func parityState(e *corentity.Entity) *mobParityState {
	if e == nil {
		return &mobParityState{}
	}
	if state, ok := mobParityStates.Load(e); ok {
		return state.(*mobParityState)
	}
	state := &mobParityState{}
	actual, _ := mobParityStates.LoadOrStore(e, state)
	return actual.(*mobParityState)
}

func tickParityCooldowns(state *mobParityState) {
	if state.primaryCooldown > 0 {
		state.primaryCooldown--
	}
	if state.secondaryCooldown > 0 {
		state.secondaryCooldown--
	}
	if state.angerTicks > 0 {
		state.angerTicks--
	}
	if state.jumpCooldown > 0 {
		state.jumpCooldown--
	}
	if state.phaseTicks > 0 {
		state.phaseTicks--
	}
}

func isParityFlyingMob(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeAllay, corentity.TypeBat, corentity.TypeBee,
		corentity.TypeBlaze, corentity.TypeGhast, corentity.TypeParrot,
		corentity.TypePhantom, corentity.TypeVex, corentity.TypeWither,
		corentity.TypeEnderDragon:
		return true
	default:
		return false
	}
}

func isParityHoppingMob(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeRabbit, corentity.TypeSlime, corentity.TypeMagmaCube, corentity.TypeBreeze:
		return true
	default:
		return false
	}
}

func isParityClimbingMob(t corentity.EntityType) bool {
	return t == corentity.TypeSpider || t == corentity.TypeCaveSpider
}

func isParityAmphibiousMob(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeAxolotl, corentity.TypeDrowned, corentity.TypeFrog, corentity.TypeTurtle:
		return true
	default:
		return false
	}
}

func parityFlightSpeed(t corentity.EntityType, requested float64) float64 {
	minimum := 0.08
	switch t {
	case corentity.TypeBat:
		minimum = 0.12
	case corentity.TypeBee, corentity.TypeParrot, corentity.TypeAllay:
		minimum = 0.10
	case corentity.TypeBlaze:
		minimum = 0.09
	case corentity.TypeGhast:
		minimum = 0.07
	case corentity.TypePhantom:
		minimum = 0.20
	case corentity.TypeVex:
		minimum = 0.18
	case corentity.TypeWither:
		minimum = 0.15
	case corentity.TypeEnderDragon:
		minimum = 0.30
	}
	if requested > minimum {
		return requested
	}
	return minimum
}

// navigateMobByParity handles navigation models that cannot use the generic
// ground A* navigator. It returns handled=true when the mob owns MOVE for this
// tick. The caller must not run ground A* after a handled result.
func (s *Server) navigateMobByParity(e *corentity.Entity, ai *mobAI, destination spatial.Vec3, speed float64) (handled, moving bool) {
	if e == nil || ai == nil {
		return false, false
	}

	if e.Type == corentity.TypeShulker {
		clearMobNavigation(e, ai)
		e.VY = 0
		return true, false
	}

	if e.Type == corentity.TypeStrider {
		return true, s.navigateStrider(e, destination, speed)
	}

	if isParityFlyingMob(e.Type) {
		return true, s.navigateFlyingMob(e, destination, parityFlightSpeed(e.Type, speed))
	}

	if isParityAmphibiousMob(e.Type) && s.entityInWater(e) {
		return true, s.navigateSwimmingMob(e, destination, speed)
	}

	if isParityHoppingMob(e.Type) {
		return true, s.navigateHoppingMob(e, destination, speed)
	}

	return false, false
}

func (s *Server) navigateFlyingMob(e *corentity.Entity, destination spatial.Vec3, speed float64) bool {
	dx := destination.X - e.Position.X
	dy := destination.Y - e.Position.Y
	dz := destination.Z - e.Position.Z
	if e.Type == corentity.TypePhantom {
		// Phantoms approach the target's upper body instead of scraping along the
		// floor like a ground mob.
		dy += 1.5
	}
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance < 0.35 {
		e.VX, e.VY, e.VZ = 0, 0, 0
		return false
	}
	e.VX = dx / distance * speed
	e.VY = dy / distance * speed
	e.VZ = dz / distance * speed
	e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	return true
}

func (s *Server) navigateSwimmingMob(e *corentity.Entity, destination spatial.Vec3, speed float64) bool {
	dx := destination.X - e.Position.X
	dy := destination.Y - e.Position.Y
	dz := destination.Z - e.Position.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance < 0.35 {
		e.VX, e.VY, e.VZ = 0, 0, 0
		return false
	}
	if speed < 0.06 {
		speed = 0.06
	}
	e.VX = dx / distance * speed
	e.VY = dy / distance * speed
	e.VZ = dz / distance * speed
	e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	return true
}

func (s *Server) navigateHoppingMob(e *corentity.Entity, destination spatial.Vec3, speed float64) bool {
	state := parityState(e)
	tickParityCooldowns(state)
	dx := destination.X - e.Position.X
	dz := destination.Z - e.Position.Z
	distance := math.Hypot(dx, dz)
	if distance < 0.35 {
		e.VX, e.VZ = 0, 0
		return false
	}
	if speed < 0.08 {
		speed = 0.08
	}
	e.VX, e.VZ = dx/distance*speed, dz/distance*speed
	e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	if e.OnGround && state.jumpCooldown <= 0 {
		switch e.Type {
		case corentity.TypeRabbit:
			e.VY = 0.42
			state.jumpCooldown = 10
		case corentity.TypeSlime:
			e.VY = 0.42
			state.jumpCooldown = 10 + int(uint32(e.EntityID)%10)
		case corentity.TypeMagmaCube:
			e.VY = 0.42
			state.jumpCooldown = 20 + int(uint32(e.EntityID)%20)
		case corentity.TypeBreeze:
			e.VY = 0.62
			state.jumpCooldown = 18
		}
	}
	return true
}

func (s *Server) navigateStrider(e *corentity.Entity, destination spatial.Vec3, speed float64) bool {
	if s.world == nil {
		return false
	}
	x, y, z := int(math.Floor(e.Position.X)), int(math.Floor(e.Position.Y)), int(math.Floor(e.Position.Z))
	feet := s.world.GetBlock(x, y, z).ResourceLocation()
	below := s.world.GetBlock(x, y-1, z).ResourceLocation()
	inLava := feet == "minecraft:lava" || below == "minecraft:lava"
	if !inLava && s.entityInWater(e) {
		// Striders shiver and are intentionally sluggish away from lava.
		speed *= 0.45
	}
	if speed < 0.04 {
		speed = 0.04
	}
	dx, dz := destination.X-e.Position.X, destination.Z-e.Position.Z
	distance := math.Hypot(dx, dz)
	if distance < 0.35 {
		e.VX, e.VZ = 0, 0
		return false
	}
	e.VX, e.VZ = dx/distance*speed, dz/distance*speed
	if inLava {
		e.VY = 0.02
	}
	e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	return true
}

// tickParityPassiveIdle owns the idle goal stack for mobs whose vanilla
// behaviour is materially different from random ground wandering. It is called
// after panic, breeding and riding have had their higher-priority chance to run.
func (s *Server) tickParityPassiveIdle(e *corentity.Entity, ai *mobAI) bool {
	if e == nil || ai == nil {
		return false
	}
	state := parityState(e)
	tickParityCooldowns(state)

	switch e.Type {
	case corentity.TypeBat:
		return s.tickBatParity(e, ai, state)
	case corentity.TypeAllay, corentity.TypeBee, corentity.TypeParrot:
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
	case corentity.TypeCat, corentity.TypeWolf, corentity.TypeParrot:
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
		// Their canonical controller is fully aquatic; avoid falling back to a
		// land wander goal when they are in water.
		if s.entityInWater(e) {
			s.tickAquaticMobAI(e, ai)
			return true
		}
	}
	return false
}

func (s *Server) tickPassiveFlightParity(e *corentity.Entity, ai *mobAI, state *mobParityState) bool {
	if e.Tamed && e.HasTameOwner {
		if s.tickTameableFollowParity(e, ai) {
			return true
		}
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
	day := ((s.worldAge%24000)+24000)%24000 < 12000
	if day && above != "minecraft:air" && above != "minecraft:water" && above != "minecraft:lava" {
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
	leader := s.closestEntityOfTypes(e, 12, e.Type)
	if leader != nil {
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
		s.navigateSwimmingMob(e, spatial.Vec3{X: target.Position.X, Y: target.Position.Y+0.5, Z: target.Position.Z}, pumpkinMovementSpeed(e.Type, 1.2))
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
		// Vanilla tameables teleport to their owner when pathing falls far behind.
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
	// Goats periodically choose a straight ram line. The cooldown is intentionally
	// long so this remains a distinct goal rather than ordinary melee pursuit.
	if distance <= 2.0 {
		if s.mobHasLineOfSight(e, target.Position, 1.62) {
			s.damagePlayerFromParityMob(target, 2, "was rammed by a goat")
			target.Position.X += dx / distance * 1.5
			target.Position.Z += dz / distance * 1.5
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

// tickParityHostileNavigationSpecials runs special attacks whose vanilla goal
// owns MOVE/LOOK at range. It is deliberately called from navigateMob because
// the legacy common hostile controller only delegates here after its generic
// melee/ranged goals decline to act.
func (s *Server) tickParityHostileNavigationSpecials(e *corentity.Entity, ai *mobAI, destination spatial.Vec3) bool {
	if e == nil {
		return false
	}
	state := parityState(e)
	tickParityCooldowns(state)
	target := s.closestPlayerToPosition(destination, 2.5)

	switch e.Type {
	case corentity.TypeGuardian, corentity.TypeElderGuardian:
		if target == nil {
			return false
		}
		distance := distance2D(e.Position, target.Position)
		if distance <= 15 && s.mobHasLineOfSight(e, target.Position, 1.62) {
			if state.phaseTicks <= 0 && state.primaryCooldown <= 0 {
				state.phaseTicks = 40
				state.primaryCooldown = 60
			} else if state.phaseTicks == 1 {
				damage := float32(6)
				if e.Type == corentity.TypeElderGuardian {
					damage = 8
				}
				s.damagePlayerFromParityMob(target, damage, "was impaled by a guardian")
			}
			e.VX, e.VZ = 0, 0
			return true
		}
	case corentity.TypeWarden:
		if target == nil {
			return false
		}
		distance := distance2D(e.Position, target.Position)
		if distance > 2.5 && distance <= 15 && state.primaryCooldown <= 0 && s.mobHasLineOfSight(e, target.Position, 1.62) {
			s.damagePlayerFromParityMob(target, 10, "was obliterated by a sonically-charged shriek")
			dx, dz := target.Position.X-e.Position.X, target.Position.Z-e.Position.Z
			if length := math.Hypot(dx, dz); length > 0.001 {
				s.sendParityPlayerVelocity(target, dx/length*2.5, 0.5, dz/length*2.5)
			}
			state.primaryCooldown = 40
			return true
		}
	case corentity.TypeShulker:
		if target != nil && state.primaryCooldown <= 0 && distance2D(e.Position, target.Position) <= 16 && s.mobHasLineOfSight(e, target.Position, 1.62) {
			// Shulker bullets need their own projectile entity/metadata path. Until
			// that adapter exists, keep the attack timing and levitation-producing
			// hit server-authoritative rather than incorrectly using ground melee.
			s.damagePlayerFromParityMob(target, 4, "was shot by a shulker")
			state.primaryCooldown = 20 + int(uint32(e.EntityID)%20)
		}
		e.VX, e.VY, e.VZ = 0, 0, 0
		return true
	case corentity.TypeEvoker:
		if target != nil && state.primaryCooldown <= 0 && distance2D(e.Position, target.Position) <= 12 {
			// Fang spell approximation is server-authoritative damage at the target
			// position. The dedicated evoker-fang entity/animation is tracked as a
			// separate adapter parity item.
			s.damagePlayerFromParityMob(target, 6, "was bitten by evocation fangs")
			state.primaryCooldown = 100
			return true
		}
	}
	return false
}

// tickOutOfBandParityMob handles mobs that were missing from the legacy
// isPassiveMob/isHostileMob classifiers entirely. It is invoked serially from
// the pre-pass so bosses are not silently motionless.
func (s *Server) tickOutOfBandParityMob(e *corentity.Entity) bool {
	if e == nil || e.Dead {
		return false
	}
	ai := s.mobAIFor(e)
	state := parityState(e)
	tickParityCooldowns(state)

	switch e.Type {
	case corentity.TypeGiant:
		target := s.closestVisiblePlayer(e, 32)
		if target == nil {
			s.tickHostileIdleGoals(e, ai)
			return true
		}
		return s.tickPlayerHunter(e, ai, state, target, 50, 20, 3.5, 0.25)
	case corentity.TypeZombifiedPiglin:
		// Neutral by default. Damage/anger integration promotes this state when
		// an attacker is known; until then it wanders instead of attacking every
		// player as a generic hostile would.
		if state.angerTicks <= 0 {
			s.tickHostileIdleGoals(e, ai)
			return true
		}
		if target := s.closestVisiblePlayer(e, 32); target != nil {
			return s.tickPlayerHunter(e, ai, state, target, 5, 20, 1.8, 0.23)
		}
		return true
	case corentity.TypeEnderDragon:
		if !state.hasAnchor || state.phaseTicks <= 0 || distanceSquaredVec(e.Position, state.anchor) < 16 {
			state.anchor = spatial.Vec3{X: e.Position.X + ai.rng.Float64()*48 - 24, Y: math.Max(70, e.Position.Y+ai.rng.Float64()*20-10), Z: e.Position.Z + ai.rng.Float64()*48 - 24}
			state.hasAnchor = true
			state.phaseTicks = 80 + ai.rng.Intn(80)
		}
		if target := s.closestVisiblePlayer(e, 64); target != nil && state.primaryCooldown <= 0 && distance2D(e.Position, target.Position) <= 5 {
			s.damagePlayerFromParityMob(target, 10, "was slain by the Ender Dragon")
			state.primaryCooldown = 10
		}
		s.navigateFlyingMob(e, state.anchor, parityFlightSpeed(e.Type, 0.30))
		return true
	}
	return false
}

func (s *Server) tickEntityHunter(attacker *corentity.Entity, ai *mobAI, state *mobParityState, target *corentity.Entity, damage float32, cooldown int, reach, speed float64) bool {
	if attacker == nil || target == nil || target.Dead {
		return false
	}
	dx, dz := target.Position.X-attacker.Position.X, target.Position.Z-attacker.Position.Z
	distance := math.Hypot(dx, dz)
	if distance <= reach {
		if state.primaryCooldown <= 0 && s.mobHasLineOfSight(attacker, target.Position, 1.0) {
			s.world.QueueEntityDamageFrom(target.EntityID, damage, attacker.Position.X, attacker.Position.Z)
			state.primaryCooldown = cooldown
		}
		attacker.VX, attacker.VZ = 0, 0
		return true
	}
	s.navigateMob(attacker, ai, target.Position, speed)
	return true
}

func (s *Server) tickPlayerHunter(attacker *corentity.Entity, ai *mobAI, state *mobParityState, target *player.Player, damage float32, cooldown int, reach, speed float64) bool {
	if attacker == nil || target == nil || target.Dead {
		return false
	}
	dx, dz := target.Position.X-attacker.Position.X, target.Position.Z-attacker.Position.Z
	distance := math.Hypot(dx, dz)
	attacker.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	if distance <= reach {
		if state.primaryCooldown <= 0 && s.mobHasLineOfSight(attacker, target.Position, 1.62) {
			s.damagePlayerFromParityMob(target, damage, "was slain by "+string(attacker.Type))
			state.primaryCooldown = cooldown
		}
		attacker.VX, attacker.VZ = 0, 0
		return true
	}
	s.navigateMob(attacker, ai, target.Position, speed)
	return true
}

func (s *Server) damagePlayerFromParityMob(target *player.Player, damage float32, cause string) bool {
	if target == nil {
		return false
	}
	var targetSession *session.Session
	if s.sessions != nil {
		if current, ok := s.sessions.Get(target.UUID); ok {
			targetSession = current
		}
	}
	if targetSession == nil {
		targetSession = &session.Session{Player: target}
	}
	return handler.DamagePlayer(targetSession, damage, cause, s.sessions)
}

func (s *Server) sendParityPlayerVelocity(target *player.Player, x, y, z float64) {
	if target == nil {
		return
	}
	var targetSession *session.Session
	if s.sessions != nil {
		if current, ok := s.sessions.Get(target.UUID); ok {
			targetSession = current
		}
	}
	if targetSession == nil {
		targetSession = &session.Session{Player: target}
	}
	s.sendPlayerVelocity(targetSession, x, y, z)
}

func (s *Server) closestEntityOfTypes(source *corentity.Entity, radius float64, types ...corentity.EntityType) *corentity.Entity {
	allowed := make(map[corentity.EntityType]struct{}, len(types))
	for _, entityType := range types {
		allowed[entityType] = struct{}{}
	}
	return s.closestEntityMatching(source, radius, func(candidate *corentity.Entity) bool {
		_, ok := allowed[candidate.Type]
		return ok
	})
}

func (s *Server) closestEntityMatching(source *corentity.Entity, radius float64, accept func(*corentity.Entity) bool) *corentity.Entity {
	if s.world == nil || source == nil {
		return nil
	}
	var closest *corentity.Entity
	best := radius * radius
	for _, candidate := range s.world.Entities.Snapshot() {
		if candidate == nil || candidate == source || candidate.Dead || !accept(candidate) {
			continue
		}
		dx, dy, dz := candidate.Position.X-source.Position.X, candidate.Position.Y-source.Position.Y, candidate.Position.Z-source.Position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance < best {
			best, closest = distance, candidate
		}
	}
	return closest
}

func (s *Server) closestPlayerToPosition(position spatial.Vec3, radius float64) *player.Player {
	if s.game == nil {
		return nil
	}
	var closest *player.Player
	best := radius * radius
	s.game.OnlinePlayers(func(candidate *player.Player) {
		if candidate == nil || candidate.Dead || candidate.GameMode == player.GameModeSpectator || candidate.Dimension != s.simulationDimension {
			return
		}
		dx, dy, dz := candidate.Position.X-position.X, candidate.Position.Y-position.Y, candidate.Position.Z-position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance < best {
			best, closest = distance, candidate
		}
	})
	return closest
}

func distance2D(a, b spatial.Vec3) float64 {
	return math.Hypot(a.X-b.X, a.Z-b.Z)
}

func distanceSquaredVec(a, b spatial.Vec3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return dx*dx + dy*dy + dz*dz
}
