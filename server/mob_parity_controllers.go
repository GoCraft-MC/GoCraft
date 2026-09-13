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

// mobParityState contains transient Goal/Brain state that is deliberately not
// persisted in entity NBT: attack cooldowns, jump cadence, anger timers and
// flight anchors. Canonical persistent state stays on core/entity.Entity.
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

// navigateMobByParity owns MOVE for navigation models that are not ground A*.
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
	dx, dy, dz := destination.X-e.Position.X, destination.Y-e.Position.Y, destination.Z-e.Position.Z
	if e.Type == corentity.TypePhantom {
		dy += 1.5
	}
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance < 0.35 {
		e.VX, e.VY, e.VZ = 0, 0, 0
		return false
	}
	e.VX, e.VY, e.VZ = dx/distance*speed, dy/distance*speed, dz/distance*speed
	e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	return true
}

func (s *Server) navigateSwimmingMob(e *corentity.Entity, destination spatial.Vec3, speed float64) bool {
	dx, dy, dz := destination.X-e.Position.X, destination.Y-e.Position.Y, destination.Z-e.Position.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance < 0.35 {
		e.VX, e.VY, e.VZ = 0, 0, 0
		return false
	}
	if speed < 0.06 {
		speed = 0.06
	}
	e.VX, e.VY, e.VZ = dx/distance*speed, dy/distance*speed, dz/distance*speed
	e.Yaw = float32(math.Atan2(-dx, dz) * 180 / math.Pi)
	return true
}

func (s *Server) navigateHoppingMob(e *corentity.Entity, destination spatial.Vec3, speed float64) bool {
	state := parityState(e)
	if state.jumpCooldown > 0 {
		state.jumpCooldown--
	}
	dx, dz := destination.X-e.Position.X, destination.Z-e.Position.Z
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
			e.VY, state.jumpCooldown = 0.42, 10
		case corentity.TypeSlime:
			e.VY, state.jumpCooldown = 0.42, 10+int(uint32(e.EntityID)%10)
		case corentity.TypeMagmaCube:
			e.VY, state.jumpCooldown = 0.42, 20+int(uint32(e.EntityID)%20)
		case corentity.TypeBreeze:
			e.VY, state.jumpCooldown = 0.62, 18
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

// tickParityHostileIdle replaces random ground wandering for flying, aquatic,
// hopping and anchored hostiles when they currently have no attack target.
func (s *Server) tickParityHostileIdle(e *corentity.Entity, ai *mobAI) bool {
	if e == nil || ai == nil {
		return false
	}
	state := parityState(e)
	if state.phaseTicks > 0 {
		state.phaseTicks--
	}
	switch {
	case e.Type == corentity.TypeShulker:
		e.VX, e.VY, e.VZ = 0, 0, 0
		return true
	case isAquaticMob(e.Type) && s.entityInWater(e):
		s.tickAquaticMobAI(e, ai)
		return true
	case isParityFlyingMob(e.Type):
		if !state.hasAnchor || state.phaseTicks <= 0 || distanceSquaredVec(e.Position, state.anchor) < 1 {
			state.anchor = spatial.Vec3{X: e.Position.X + ai.rng.Float64()*20 - 10, Y: e.Position.Y + ai.rng.Float64()*10 - 5, Z: e.Position.Z + ai.rng.Float64()*20 - 10}
			state.hasAnchor = true
			state.phaseTicks = 30 + ai.rng.Intn(60)
		}
		s.navigateFlyingMob(e, state.anchor, parityFlightSpeed(e.Type, pumpkinMovementSpeed(e.Type, 0.8)))
		return true
	case isParityHoppingMob(e.Type):
		if !ai.hasWanderGoal || state.phaseTicks <= 0 {
			ai.wanderTarget = spatial.Vec3{X: e.Position.X + ai.rng.Float64()*12 - 6, Y: e.Position.Y, Z: e.Position.Z + ai.rng.Float64()*12 - 6}
			ai.hasWanderGoal = true
			state.phaseTicks = 30 + ai.rng.Intn(50)
		}
		s.navigateHoppingMob(e, ai.wanderTarget, pumpkinMovementSpeed(e.Type, 0.8))
		return true
	}
	return false
}

// tickParityHostileNavigationSpecials handles ranged/special goals that the
// old common hostile controller does not understand. Returning true means the
// special goal owns MOVE/LOOK for this tick.
func (s *Server) tickParityHostileNavigationSpecials(e *corentity.Entity, ai *mobAI, destination spatial.Vec3) bool {
	if e == nil {
		return false
	}
	switch e.Type {
	case corentity.TypeGuardian, corentity.TypeElderGuardian, corentity.TypeWarden, corentity.TypeShulker, corentity.TypeEvoker:
	default:
		return false
	}

	state := parityState(e)
	if state.primaryCooldown > 0 {
		state.primaryCooldown--
	}
	if state.phaseTicks > 0 {
		state.phaseTicks--
	}
	target := s.closestPlayerToPosition(destination, 2.5)

	switch e.Type {
	case corentity.TypeGuardian, corentity.TypeElderGuardian:
		if target == nil {
			return false
		}
		if distance2D(e.Position, target.Position) <= 15 && s.mobHasLineOfSight(e, target.Position, 1.62) {
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
			s.damagePlayerFromParityMob(target, 4, "was shot by a shulker")
			state.primaryCooldown = 20 + int(uint32(e.EntityID)%20)
		}
		e.VX, e.VY, e.VZ = 0, 0, 0
		return true
	case corentity.TypeEvoker:
		if target != nil && state.primaryCooldown <= 0 && distance2D(e.Position, target.Position) <= 12 {
			s.damagePlayerFromParityMob(target, 6, "was bitten by evocation fangs")
			state.primaryCooldown = 100
			return true
		}
	}
	return false
}

// tickOutOfBandParityMob covers mobs absent from the legacy passive/hostile
// classifiers. The pre-pass calls this serially before worker AI begins.
func (s *Server) tickOutOfBandParityMob(e *corentity.Entity) bool {
	if e == nil || e.Dead {
		return false
	}
	ai := s.mobAIFor(e)
	state := parityState(e)
	if state.primaryCooldown > 0 {
		state.primaryCooldown--
	}
	if state.angerTicks > 0 {
		state.angerTicks--
	}
	if state.phaseTicks > 0 {
		state.phaseTicks--
	}

	switch e.Type {
	case corentity.TypeGiant:
		if target := s.closestVisiblePlayer(e, 32); target != nil {
			return s.tickPlayerHunter(e, ai, state, target, 50, 20, 3.5, 0.25)
		}
		s.tickHostileIdleGoals(e, ai)
		return true
	case corentity.TypeZombifiedPiglin:
		if state.angerTicks > 0 {
			if target := s.closestVisiblePlayer(e, 32); target != nil {
				return s.tickPlayerHunter(e, ai, state, target, 5, 20, 1.8, 0.23)
			}
		}
		s.tickHostileIdleGoals(e, ai)
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
	if state.primaryCooldown > 0 {
		state.primaryCooldown--
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
	if state.primaryCooldown > 0 {
		state.primaryCooldown--
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
