package server

import (
	"math"
	"sync"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

// villagerActivity mirrors the activity set registered by Mojang's 1.21.4
// Villager.registerBrainGoals. CORE always runs; one non-core activity is
// selected by panic/bell/raid state or the villager schedule.
type villagerActivity uint8

const (
	villagerActivityIdle villagerActivity = iota
	villagerActivityWork
	villagerActivityPlay
	villagerActivityRest
	villagerActivityMeet
	villagerActivityPanic
	villagerActivityPreRaid
	villagerActivityRaid
	villagerActivityHide
)

// These names intentionally match the MemoryModuleType values in the 1.21.4
// Villager.MEMORY_TYPES list. Some memories map directly onto canonical GoCraft
// entity/mobAI fields; the rest live in villagerBrainState below.
var villagerMemoryModules = [...]string{
	"home",
	"job_site",
	"potential_job_site",
	"meeting_point",
	"nearest_living_entities",
	"nearest_visible_living_entities",
	"visible_villager_babies",
	"nearest_players",
	"nearest_visible_player",
	"nearest_visible_attackable_player",
	"nearest_visible_wanted_item",
	"item_pickup_cooldown_ticks",
	"walk_target",
	"look_target",
	"interaction_target",
	"breed_target",
	"path",
	"doors_to_close",
	"nearest_bed",
	"hurt_by",
	"hurt_by_entity",
	"nearest_hostile",
	"secondary_job_site",
	"hiding_place",
	"heard_bell_time",
	"cant_reach_walk_target_since",
	"last_slept",
	"last_woken",
	"last_worked_at_poi",
	"golem_detected_recently",
}

var villagerSensorTypes = [...]string{
	"nearest_living_entities",
	"nearest_players",
	"nearest_items",
	"nearest_bed",
	"hurt_by",
	"villager_hostiles",
	"villager_babies",
	"secondary_pois",
	"golem_detected",
}

type villagerBrainKey struct {
	server   *Server
	entityID int32
}

// villagerBrainState stores the 1.21.4 Brain memories that do not already have
// a canonical home in Entity or mobAI. Spatial POI memories intentionally stay
// in Entity so Java and Bedrock share exactly one source of truth.
type villagerBrainState struct {
	activity villagerActivity

	potentialJobSite    spatial.BlockPos
	hasPotentialJobSite bool
	nearestBed          spatial.BlockPos
	hasNearestBed       bool
	hidingPlace         spatial.BlockPos
	hasHidingPlace      bool
	secondaryJobSites   []spatial.BlockPos

	nearestHostileID   int32
	nearestBabyID      int32
	nearestWantedItem  int32
	interactionTarget  int32
	itemPickupCooldown int

	heardBellTime            int64
	hasHeardBellTime         bool
	cantReachWalkTargetSince int64
	hasCantReachSince        bool
	lastSlept                int64
	lastWoken                int64
	lastWorkedAtPOI          int64
	golemDetectedUntil       int64

	lastSensorTick int64
	lastSeenTick   int64
	playRetargetAt int64
	workActionAt   int64
}

var villagerBrainStates = struct {
	sync.Mutex
	values map[villagerBrainKey]*villagerBrainState
}{values: make(map[villagerBrainKey]*villagerBrainState)}

func (s *Server) villagerBrainStateFor(villager *corentity.Entity) *villagerBrainState {
	key := villagerBrainKey{server: s, entityID: villager.EntityID}
	villagerBrainStates.Lock()
	state := villagerBrainStates.values[key]
	if state == nil {
		state = &villagerBrainState{}
		villagerBrainStates.values[key] = state
	}
	state.lastSeenTick = s.worldAge
	villagerBrainStates.Unlock()
	return state
}

func (s *Server) cleanupVillagerBrainStates() {
	if s == nil || s.worldAge%1200 != 0 {
		return
	}
	villagerBrainStates.Lock()
	for key, state := range villagerBrainStates.values {
		if key.server == s && s.worldAge-state.lastSeenTick > 2400 {
			delete(villagerBrainStates.values, key)
		}
	}
	villagerBrainStates.Unlock()
}

// tickVillagerBrain is the serial Brain phase. It is deliberately executed by
// tickVillagerDoor before passive workers start: sensors inspect shared entity
// state, while the worker phase only consumes the resulting per-villager MOVE
// and LOOK targets.
func (s *Server) tickVillagerBrain(villager *corentity.Entity, ai *mobAI) {
	if s == nil || s.world == nil || villager == nil || ai == nil || villager.Dead {
		return
	}
	state := s.villagerBrainStateFor(villager)
	s.cleanupVillagerBrainStates()
	s.tickVillagerSensors(villager, ai, state)
	s.validateVillagerPOIMemories(villager, state)

	activity := s.selectVillagerActivity(villager, ai, state)
	state.activity = activity

	// CORE package behaviours which affect movement/state independent of the
	// scheduled activity. Door interaction and MoveToTargetSink are handled by
	// tickVillagerDoor/navigateMob. Profession acquisition/reset is backed by
	// the canonical world POI registry and refreshed by the server every second.
	s.applyVillagerCoreBehaviors(villager, ai, state)

	switch activity {
	case villagerActivityPanic:
		s.applyVillagerPanic(villager, ai, state)
	case villagerActivityHide:
		s.applyVillagerHide(villager, ai, state)
	case villagerActivityPreRaid:
		s.applyVillagerPreRaid(villager, ai, state)
	case villagerActivityRaid:
		s.applyVillagerRaid(villager, ai, state)
	case villagerActivityWork:
		s.applyVillagerWork(villager, ai, state)
	case villagerActivityMeet:
		s.applyVillagerMeet(villager, ai, state)
	case villagerActivityPlay:
		s.applyVillagerPlay(villager, ai, state)
	case villagerActivityRest:
		s.applyVillagerRest(villager, ai, state)
	default:
		s.applyVillagerIdle(villager, ai, state)
	}
}

func (s *Server) tickVillagerSensors(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	// Vanilla sensors have their own scan rates. Twenty ticks keeps GoCraft's
	// entity snapshot work bounded while matching the cadence used by the POI
	// refresh loop; hurt/panic state itself is still consumed every tick.
	if state.lastSensorTick != 0 && s.worldAge-state.lastSensorTick < 20 {
		return
	}
	state.lastSensorTick = s.worldAge
	state.nearestHostileID = 0
	state.nearestBabyID = 0
	state.nearestWantedItem = 0
	state.hasNearestBed = false
	state.secondaryJobSites = state.secondaryJobSites[:0]

	bestHostile, bestBaby, bestItem := 16.0*16.0, 16.0*16.0, 32.0*32.0
	golemNearby := false
	for _, candidate := range s.world.Entities.Snapshot() {
		if candidate == nil || candidate == villager || candidate.Dead {
			continue
		}
		dx := candidate.Position.X - villager.Position.X
		dy := candidate.Position.Y - villager.Position.Y
		dz := candidate.Position.Z - villager.Position.Z
		distanceSquared := dx*dx + dy*dy + dz*dz
		switch {
		case isHostileMob(candidate.Type) && distanceSquared < bestHostile:
			if s.mobHasLineOfSight(villager, candidate.Position, 1.4) {
				bestHostile = distanceSquared
				state.nearestHostileID = candidate.EntityID
			}
		case candidate.Type == corentity.TypeVillager && candidate.IsBaby && distanceSquared < bestBaby:
			if s.mobHasLineOfSight(villager, candidate.Position, 1.0) {
				bestBaby = distanceSquared
				state.nearestBabyID = candidate.EntityID
			}
		case candidate.Type == corentity.TypeItem && villagerWantsItem(villager, candidate.ItemID) && distanceSquared < bestItem:
			// Paper's MC-157464 fix: sleeping villagers must not acquire a wanted
			// item walk target. Keep the memory absent while sleeping.
			if !villager.Sleeping && s.mobHasLineOfSight(villager, candidate.Position, 0.25) {
				bestItem = distanceSquared
				state.nearestWantedItem = candidate.EntityID
			}
		case candidate.Type == corentity.TypeIronGolem && distanceSquared <= 16*16:
			golemNearby = true
		}
	}
	if golemNearby {
		state.golemDetectedUntil = s.worldAge + 600
	}

	origin := spatial.BlockPos{X: int32(math.Floor(villager.Position.X)), Y: int32(math.Floor(villager.Position.Y)), Z: int32(math.Floor(villager.Position.Z))}
	beds := s.world.VillageBedsNear(origin, 48)
	bestBedDistance := math.MaxFloat64
	for _, bed := range beds {
		dx := float64(bed.X) + 0.5 - villager.Position.X
		dy := float64(bed.Y) - villager.Position.Y
		dz := float64(bed.Z) + 0.5 - villager.Position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance < bestBedDistance {
			bestBedDistance = distance
			state.nearestBed = bed
			state.hasNearestBed = true
		}
	}

	if villager.VillagerProfession == corentity.VillagerProfessionFarmer {
		// Farmer is the only vanilla profession with secondary POIs in 1.21.4:
		// nearby farmland blocks.
		cx, cy, cz := int(math.Floor(villager.Position.X)), int(math.Floor(villager.Position.Y)), int(math.Floor(villager.Position.Z))
		for x := cx - 8; x <= cx+8 && len(state.secondaryJobSites) < 16; x++ {
			for y := cy - 2; y <= cy+2 && len(state.secondaryJobSites) < 16; y++ {
				for z := cz - 8; z <= cz+8 && len(state.secondaryJobSites) < 16; z++ {
					block, loaded := s.world.BlockIfLoaded(x, y, z)
					if loaded && block.ResourceLocation() == "minecraft:farmland" {
						state.secondaryJobSites = append(state.secondaryJobSites, spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)})
					}
				}
			}
		}
	}
}

func (s *Server) validateVillagerPOIMemories(villager *corentity.Entity, state *villagerBrainState) {
	if villager.HasVillageHome && !s.validVillagerBed(villager) {
		villager.HasVillageHome = false
		villager.VillageBed = spatial.BlockPos{}
	}
	if villager.HasVillageWorkstation {
		job := villager.VillageWorkstation
		block, loaded := s.world.BlockIfLoaded(int(job.X), int(job.Y), int(job.Z))
		if loaded && block.IsAir() {
			villager.HasVillageWorkstation = false
			villager.VillageWorkstation = spatial.BlockPos{}
			if !villager.VillagerHasTraded {
				villager.VillagerProfession = corentity.VillagerProfessionNone
			}
		}
	}
	if villager.VillageCenter == (spatial.BlockPos{}) {
		if bell, ok := s.nearestVillagerMeetingPoint(villager, 32); ok {
			villager.VillageCenter = bell
		}
	}
	if state.hasPotentialJobSite {
		block, loaded := s.world.BlockIfLoaded(int(state.potentialJobSite.X), int(state.potentialJobSite.Y), int(state.potentialJobSite.Z))
		if loaded && block.IsAir() {
			state.hasPotentialJobSite = false
		}
	}
}

func (s *Server) selectVillagerActivity(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) villagerActivity {
	if ai.panicTick > 0 || state.nearestHostileID != 0 {
		return villagerActivityPanic
	}
	if state.hasHeardBellTime && s.worldAge-state.heardBellTime < 300 {
		// GoCraft has no raid object at this layer. A bell with no active raid is
		// exactly the vanilla ReactToBell path that activates HIDE.
		return villagerActivityHide
	}
	dayTime := s.worldAge % 24000
	if dayTime < 0 {
		dayTime += 24000
	}
	if villager.IsBaby {
		switch {
		case dayTime >= 12000:
			return villagerActivityRest
		case dayTime >= 10000:
			return villagerActivityPlay
		case dayTime >= 6000:
			return villagerActivityIdle
		case dayTime >= 3000:
			return villagerActivityPlay
		default:
			return villagerActivityIdle
		}
	}
	switch {
	case dayTime >= 12000:
		return villagerActivityRest
	case dayTime >= 11000:
		return villagerActivityIdle
	case dayTime >= 9000:
		if villager.VillageCenter != (spatial.BlockPos{}) {
			return villagerActivityMeet
		}
		return villagerActivityIdle
	case dayTime >= 2000:
		if villager.HasVillageWorkstation {
			return villagerActivityWork
		}
		return villagerActivityIdle
	default:
		return villagerActivityIdle
	}
}

func (s *Server) applyVillagerCoreBehaviors(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	// Acquire HOME. tickVillagerBedClaim performs the actual loaded-chunk scan
	// immediately after this brain phase; use NEAREST_BED to encourage movement
	// toward a candidate while the claim memory is absent.
	if !villager.HasVillageHome && state.hasNearestBed && state.activity == villagerActivityRest {
		target := state.nearestBed
		s.setVillagerWalkTarget(villager, ai, spatial.Vec3{X: float64(target.X) + 0.5, Y: float64(target.Y), Z: float64(target.Z) + 0.5})
	}

	// Acquire/assign/reset profession/job-site semantics are centralized in the
	// world POI registry. Running the refresh here makes the Brain responsive on
	// the same tick instead of waiting for the one-second server maintenance pass.
	if !villager.IsBaby && s.worldAge%20 == int64(uint32(villager.EntityID)%20) {
		for _, changed := range s.world.RefreshVillagerProfessions(10) {
			handler.BroadcastVillagerMetadata(changed, s.sessions)
		}
	}

	// GoToWantedItem is a CORE behaviour. Do not steal MOVE from panic/hide,
	// work/meet/rest targets; in IDLE/PLAY it can acquire the item target.
	if state.nearestWantedItem != 0 && state.itemPickupCooldown <= 0 && !villager.Sleeping &&
		(state.activity == villagerActivityIdle || state.activity == villagerActivityPlay) {
		if item, ok := s.world.Entities.Get(state.nearestWantedItem); ok && item.Type == corentity.TypeItem && !item.Dead {
			s.setVillagerWalkTarget(villager, ai, item.Position)
			state.interactionTarget = item.EntityID
		}
	}
	if state.itemPickupCooldown > 0 {
		state.itemPickupCooldown--
	}
}

func (s *Server) applyVillagerWork(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if !villager.HasVillageWorkstation {
		return
	}
	job := villager.VillageWorkstation
	target := spatial.Vec3{X: float64(job.X) + 0.5, Y: float64(job.Y), Z: float64(job.Z) + 0.5}
	s.setVillagerWalkTarget(villager, ai, target)
	// Scheduled POI movement must survive tickPassiveMobAI's generic home-bound
	// branch. Vanilla's WALK_TARGET owns MOVE here, so temporarily allow this
	// target to be consumed by the shared navigator.
	ai.roaming = true

	dx, dz := villager.Position.X-target.X, villager.Position.Z-target.Z
	if dx*dx+dz*dz <= 2.5*2.5 {
		state.lastWorkedAtPOI = s.worldAge
		if villager.VillagerProfession == corentity.VillagerProfessionFarmer && s.worldAge >= state.workActionAt {
			s.tickFarmerVillagerWork(villager, state)
			state.workActionAt = s.worldAge + 20
		}
	}
}

func (s *Server) applyVillagerMeet(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	center := villager.VillageCenter
	if center == (spatial.BlockPos{}) {
		return
	}
	angle := float64(uint32(villager.EntityID)%16) * (2 * math.Pi / 16)
	radius := 2.0 + float64(uint32(villager.EntityID)%3)*0.5
	target := spatial.Vec3{
		X: float64(center.X) + 0.5 + math.Cos(angle)*radius,
		Y: float64(center.Y),
		Z: float64(center.Z) + 0.5 + math.Sin(angle)*radius,
	}
	s.setVillagerWalkTarget(villager, ai, target)
	ai.roaming = true

	// SocializeAtBell / TradeWithVillager: once at the meeting point, look at a
	// nearby villager and keep an interaction memory. Actual merchant offer
	// exchange remains owned by the canonical trade subsystem.
	if targetVillager := s.nearestVillager(villager, 5, false); targetVillager != nil {
		state.interactionTarget = targetVillager.EntityID
		ai.lookX, ai.lookZ = targetVillager.Position.X, targetVillager.Position.Z
		ai.lookTick = 20
	}
}

func (s *Server) applyVillagerIdle(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	// IDLE's RunOne contains villager/cat interaction, breeding interaction,
	// village-bound strolling, look-target walking, bed jumping and DoNothing.
	// The shared passive idle selector already supplies stroll/look/do-nothing;
	// add the villager social/breed target memory here.
	if target := s.nearestVillager(villager, 8, true); target != nil {
		state.interactionTarget = target.EntityID
		if !villager.IsBaby && !target.IsBaby && villager.BreedingMateEntityID == 0 {
			// This maps BREED_TARGET. The actual vanilla food/bed-gated birth path
			// is kept separate from generic animal LoveTicks.
			villager.BreedingMateEntityID = target.EntityID
		}
	}
}

func (s *Server) applyVillagerPlay(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if !villager.IsBaby {
		return
	}
	if state.nearestBabyID != 0 {
		if other, ok := s.world.Entities.Get(state.nearestBabyID); ok && other.Type == corentity.TypeVillager && other.IsBaby && !other.Dead {
			dx, dz := other.Position.X-villager.Position.X, other.Position.Z-villager.Position.Z
			distance := math.Hypot(dx, dz)
			if distance > 2 {
				target := other.Position
				// PlayTagWithOtherKids does not stack children on the same block.
				if distance > 0.001 {
					target.X -= dx / distance * 1.5
					target.Z -= dz / distance * 1.5
				}
				s.setVillagerWalkTarget(villager, ai, target)
				ai.roaming = true
				return
			}
		}
	}
	if s.worldAge >= state.playRetargetAt {
		state.playRetargetAt = s.worldAge + int64(40+ai.rng.Intn(40))
		center := villager.VillageCenter
		baseX, baseZ := villager.Position.X, villager.Position.Z
		if center != (spatial.BlockPos{}) {
			baseX, baseZ = float64(center.X)+0.5, float64(center.Z)+0.5
		}
		target := spatial.Vec3{X: baseX + ai.rng.Float64()*12 - 6, Y: villager.Position.Y, Z: baseZ + ai.rng.Float64()*12 - 6}
		s.setVillagerWalkTarget(villager, ai, target)
		ai.roaming = true
	}
}

func (s *Server) applyVillagerRest(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if villager.Sleeping {
		state.lastSlept = s.worldAge
		return
	}
	if villager.HasVillageHome && s.validVillagerBed(villager) {
		bed := villager.VillageBed
		s.setVillagerWalkTarget(villager, ai, spatial.Vec3{X: float64(bed.X) + 0.5, Y: float64(bed.Y), Z: float64(bed.Z) + 0.5})
		ai.roaming = true
		return
	}
	if state.hasNearestBed {
		bed := state.nearestBed
		s.setVillagerWalkTarget(villager, ai, spatial.Vec3{X: float64(bed.X) + 0.5, Y: float64(bed.Y), Z: float64(bed.Z) + 0.5})
		ai.roaming = true
	}
}

func (s *Server) applyVillagerPanic(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if state.nearestHostileID == 0 {
		return
	}
	hostile, ok := s.world.Entities.Get(state.nearestHostileID)
	if !ok || hostile.Dead {
		return
	}
	dx := villager.Position.X - hostile.Position.X
	dz := villager.Position.Z - hostile.Position.Z
	distance := math.Hypot(dx, dz)
	if distance < 0.001 {
		angle := ai.rng.Float64() * math.Pi * 2
		dx, dz, distance = math.Cos(angle), math.Sin(angle), 1
	}
	ai.dirX, ai.dirZ = dx/distance, dz/distance
	ai.targetX = villager.Position.X + ai.dirX*6
	ai.targetZ = villager.Position.Z + ai.dirZ*6
	if ai.panicTick < 40 {
		ai.panicTick = 40
	}
	villager.Sleeping = false
}

func (s *Server) applyVillagerHide(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if !state.hasHidingPlace {
		if villager.HasVillageHome && s.validVillagerBed(villager) {
			state.hidingPlace = villager.VillageBed
			state.hasHidingPlace = true
		} else if state.hasNearestBed {
			state.hidingPlace = state.nearestBed
			state.hasHidingPlace = true
		}
	}
	if !state.hasHidingPlace {
		return
	}
	hide := state.hidingPlace
	s.setVillagerWalkTarget(villager, ai, spatial.Vec3{X: float64(hide.X) + 0.5, Y: float64(hide.Y), Z: float64(hide.Z) + 0.5})
	ai.roaming = true
	if s.worldAge-state.heardBellTime >= 300 {
		state.hasHeardBellTime = false
		state.hasHidingPlace = false
	}
}

// Raid activities are present in the Mojang Brain even when no raid is active.
// GoCraft's current world model has no Raid object, so these functions preserve
// the activity slots and movement contracts without fabricating raid state.
func (s *Server) applyVillagerPreRaid(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if villager.VillageCenter != (spatial.BlockPos{}) {
		center := villager.VillageCenter
		s.setVillagerWalkTarget(villager, ai, spatial.Vec3{X: float64(center.X) + 0.5, Y: float64(center.Y), Z: float64(center.Z) + 0.5})
		ai.roaming = true
	}
}

func (s *Server) applyVillagerRaid(villager *corentity.Entity, ai *mobAI, state *villagerBrainState) {
	if state.hasHidingPlace {
		s.applyVillagerHide(villager, ai, state)
	}
}

func (s *Server) setVillagerWalkTarget(villager *corentity.Entity, ai *mobAI, target spatial.Vec3) {
	goal := spatial.BlockPos{X: int32(math.Floor(target.X)), Y: int32(math.Floor(target.Y)), Z: int32(math.Floor(target.Z))}
	if ai.hasPathGoal && ai.pathGoal != goal {
		clearMobNavigation(villager, ai)
	}
	ai.wanderTarget = target
	ai.hasWanderGoal = true
}

func (s *Server) nearestVillager(villager *corentity.Entity, maximumDistance float64, excludeBabies bool) *corentity.Entity {
	var nearest *corentity.Entity
	best := maximumDistance * maximumDistance
	for _, candidate := range s.world.Entities.Snapshot() {
		if candidate == nil || candidate == villager || candidate.Dead || candidate.Type != corentity.TypeVillager || (excludeBabies && candidate.IsBaby) {
			continue
		}
		dx, dy, dz := candidate.Position.X-villager.Position.X, candidate.Position.Y-villager.Position.Y, candidate.Position.Z-villager.Position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance < best && s.mobHasLineOfSight(villager, candidate.Position, 1.4) {
			best = distance
			nearest = candidate
		}
	}
	return nearest
}

func (s *Server) nearestVillagerMeetingPoint(villager *corentity.Entity, radius int) (spatial.BlockPos, bool) {
	cx, cy, cz := int(math.Floor(villager.Position.X)), int(math.Floor(villager.Position.Y)), int(math.Floor(villager.Position.Z))
	best := math.MaxFloat64
	var result spatial.BlockPos
	found := false
	for x := cx - radius; x <= cx+radius; x++ {
		for y := cy - 4; y <= cy+4; y++ {
			for z := cz - radius; z <= cz+radius; z++ {
				block, loaded := s.world.BlockIfLoaded(x, y, z)
				if !loaded || block.ResourceLocation() != "minecraft:bell" {
					continue
				}
				dx, dy, dz := float64(x)+0.5-villager.Position.X, float64(y)-villager.Position.Y, float64(z)+0.5-villager.Position.Z
				distance := dx*dx + dy*dy + dz*dz
				if distance < best {
					best = distance
					result = spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)}
					found = true
				}
			}
		}
	}
	return result, found
}

func villagerWantsItem(villager *corentity.Entity, itemID string) bool {
	switch itemID {
	case "minecraft:bread", "minecraft:potato", "minecraft:carrot", "minecraft:beetroot":
		return true
	case "minecraft:wheat", "minecraft:wheat_seeds", "minecraft:beetroot_seeds", "minecraft:bone_meal":
		return villager != nil && villager.VillagerProfession == corentity.VillagerProfessionFarmer
	default:
		return false
	}
}

func (s *Server) tickFarmerVillagerWork(villager *corentity.Entity, state *villagerBrainState) {
	cx, cy, cz := int(math.Floor(villager.Position.X)), int(math.Floor(villager.Position.Y)), int(math.Floor(villager.Position.Z))
	for x := cx - 2; x <= cx+2; x++ {
		for y := cy - 1; y <= cy+1; y++ {
			for z := cz - 2; z <= cz+2; z++ {
				block, loaded := s.world.BlockIfLoaded(x, y, z)
				if !loaded || !villagerCropMature(block) {
					continue
				}
				block.Properties = copyStringMap(block.Properties)
				block.Properties["age"] = "0"
				s.world.SetBlock(x, y, z, block)
				handler.BroadcastBlockChange(coreworld.BlockChange{X: x, Y: y, Z: z, Block: block}, s.javaSessionsForDimension(s.simulationDimension))
				return
			}
		}
	}
}

func villagerCropMature(block coreworld.Block) bool {
	age := block.Properties["age"]
	switch block.ResourceLocation() {
	case "minecraft:wheat", "minecraft:carrots", "minecraft:potatoes":
		return age == "7"
	case "minecraft:beetroots":
		return age == "3"
	default:
		return false
	}
}

func (s *Server) notifyVillagersOfBell(world *coreworld.World, position spatial.BlockPos) {
	if s == nil || world == nil {
		return
	}
	for _, villager := range world.Entities.Snapshot() {
		if villager == nil || villager.Dead || villager.Type != corentity.TypeVillager {
			continue
		}
		dx := villager.Position.X - (float64(position.X) + 0.5)
		dy := villager.Position.Y - (float64(position.Y) + 0.5)
		dz := villager.Position.Z - (float64(position.Z) + 0.5)
		if dx*dx+dy*dy+dz*dz > 32*32 {
			continue
		}
		state := s.villagerBrainStateFor(villager)
		state.heardBellTime = s.worldAge
		state.hasHeardBellTime = true
		state.hasHidingPlace = false
	}
}
