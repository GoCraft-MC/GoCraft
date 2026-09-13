package server

import (
	"math"
	"strconv"
	"strings"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

const (
	villagerDoorCloseTicks = 60
	// LivingEntity.startSleeping(BlockPos) moves the entity onto the bed at
	// block centre with the vanilla bed-height offset. Keeping the canonical
	// server position there is important for both Java and Bedrock: the tracked
	// sleeping position controls pose/orientation, while movement packets still
	// use the entity's real position.
	villagerBedSleepYOffset = 0.6875

	// Vanilla adult villager schedule boundaries. REST is handled by the sleep
	// branch in tickPassiveMobAI; these phases feed the same navigation stack
	// with the appropriate POI target before generic idle wandering runs.
	villagerWorkStart = int64(2000)
	villagerMeetStart = int64(9000)
	villagerIdleStart = int64(11000)
	villagerRestStart = int64(12000)
)

// tickVillagerDoor mirrors the wooden-door capability used by Pumpkin's walk
// node evaluator. It also runs the villager Brain serial phase before passive
// AI workers consume WALK_TARGET/LOOK_TARGET state.
func (s *Server) tickVillagerDoor(villager *corentity.Entity, ai *mobAI) {
	if villager == nil || ai == nil {
		return
	}

	// A villager may acquire or lose HOME after mobAIFor created its state.
	// Keep the cached homing fields synchronized instead of leaving a newly
	// housed villager permanently in the roaming-animal mode.
	s.syncVillagerHomeAI(villager, ai)

	// Vanilla SleepInBed ultimately calls LivingEntity.startSleeping, which
	// records the sleeping BlockPos and repositions the entity onto the bed.
	// GoCraft already tracked the bed position/pose but previously skipped the
	// reposition, so villagers could visibly sleep on the floor next to a bed.
	if villager.Sleeping {
		s.positionVillagerInBed(villager, ai)
	} else {
		s.tickVillagerBrain(villager, ai)
	}

	if ai.doorCloseTick > 0 {
		ai.doorCloseTick--
		if ai.doorCloseTick == 0 {
			if s.villagerNearDoor(ai.openedDoor, 1.75) {
				ai.doorCloseTick = 20
			} else {
				s.setVillagerDoorOpen(ai.openedDoor, false)
				ai.openedDoor = spatial.BlockPos{}
			}
		}
	}
	if villager.Sleeping {
		return
	}
	// Vanilla InteractWithDoor opens the door on both the path's previous and
	// next node, so the door is opened as the villager approaches rather than
	// only once it is standing on the door tile. Scan the current and upcoming
	// waypoint and open any wooden door within reach; this also survives a
	// partial path whose exact node granularity does not land on the door.
	for i := ai.pathIndex; i < len(ai.path) && i <= ai.pathIndex+1; i++ {
		waypoint := ai.path[i]
		position, door, ok := s.villagerDoorAt(int(math.Floor(waypoint.X)), int(math.Floor(waypoint.Y)), int(math.Floor(waypoint.Z)))
		if !ok || door.Properties["open"] == "true" {
			continue
		}
		dx, dz := waypoint.X-villager.Position.X, waypoint.Z-villager.Position.Z
		if dx*dx+dz*dz > 2.0*2.0 {
			continue
		}
		if s.setVillagerDoorOpen(position, true) {
			ai.openedDoor = position
			ai.doorCloseTick = villagerDoorCloseTicks
		}
	}
}

// syncVillagerHomeAI mirrors the Brain HOME-memory effect on navigation. The
// entity owns the canonical POI state; mobAI only caches a movement anchor, so
// the cache must follow claims made after the AI object was first allocated.
func (s *Server) syncVillagerHomeAI(villager *corentity.Entity, ai *mobAI) {
	if villager.HasVillageHome {
		ai.roaming = false
		center := villager.VillageCenter
		if center == (spatial.BlockPos{}) {
			center = villager.VillageBed
		}
		ai.homeX = float64(center.X) + 0.5
		ai.homeZ = float64(center.Z) + 0.5
		return
	}
	ai.roaming = true
}

// applyVillagerScheduledWalkTarget is retained for compatibility with older
// tests/helpers. The complete schedule now lives in tickVillagerBrain.
func (s *Server) applyVillagerScheduledWalkTarget(villager *corentity.Entity, ai *mobAI) {
	if s == nil || villager == nil || ai == nil || villager.Sleeping {
		return
	}
	s.tickVillagerBrain(villager, ai)
}

// positionVillagerInBed is the server-side equivalent of vanilla
// LivingEntity.startSleeping/setPosToBed for villagers. HOME/bed POIs are
// expected to refer to the head half. Older generated/persisted GoCraft worlds
// may still point at a foot half, so normalise that first and keep VillageBed
// canonical for Java sleeping-position metadata and Bedrock BedPosition data.
func (s *Server) positionVillagerInBed(villager *corentity.Entity, ai *mobAI) bool {
	if s == nil || s.world == nil || villager == nil || villager.Type != corentity.TypeVillager || !villager.HasVillageHome {
		return false
	}

	bedPos, _, ok := s.villagerBedHead(villager.VillageBed)
	if !ok {
		return false
	}
	villager.VillageBed = bedPos
	villager.Position.X = float64(bedPos.X) + 0.5
	villager.Position.Y = float64(bedPos.Y) + villagerBedSleepYOffset
	villager.Position.Z = float64(bedPos.Z) + 0.5
	villager.VX, villager.VY, villager.VZ = 0, 0, 0
	villager.OnGround = true
	clearMobNavigation(villager, ai)
	return true
}

// villagerBedHead resolves either half of a bed to its head block. Vanilla
// HOME POIs are represented by the bed head; normalising here also prevents a
// persisted foot-half claim from producing the wrong sleeping anchor.
func (s *Server) villagerBedHead(position spatial.BlockPos) (spatial.BlockPos, coreworld.Block, bool) {
	if s == nil || s.world == nil {
		return spatial.BlockPos{}, coreworld.Block{}, false
	}
	x, y, z := int(position.X), int(position.Y), int(position.Z)
	bed := s.world.GetBlock(x, y, z)
	if !strings.HasSuffix(bed.ResourceLocation(), "_bed") {
		return spatial.BlockPos{}, coreworld.Block{}, false
	}
	if bed.Properties["part"] != "foot" {
		return position, bed, true
	}

	dx, dz := villagerBedFacingOffset(bed.Properties["facing"])
	headPos := spatial.BlockPos{X: position.X + int32(dx), Y: position.Y, Z: position.Z + int32(dz)}
	head := s.world.GetBlock(int(headPos.X), int(headPos.Y), int(headPos.Z))
	if head.ResourceLocation() != bed.ResourceLocation() || head.Properties["part"] != "head" {
		return spatial.BlockPos{}, coreworld.Block{}, false
	}
	return headPos, head, true
}

func villagerBedFacingOffset(facing string) (dx, dz int) {
	switch facing {
	case "north":
		return 0, -1
	case "south":
		return 0, 1
	case "west":
		return -1, 0
	case "east":
		return 1, 0
	default:
		return 0, 0
	}
}

func (s *Server) villagerDoorAt(x, y, z int) (spatial.BlockPos, coreworld.Block, bool) {
	door := s.world.GetBlock(x, y, z)
	if !isVillagerWoodenDoor(door) {
		return spatial.BlockPos{}, coreworld.Block{}, false
	}
	if door.Properties["half"] == "upper" {
		y--
		door = s.world.GetBlock(x, y, z)
	}
	return spatial.BlockPos{X: int32(x), Y: int32(y), Z: int32(z)}, door, isVillagerWoodenDoor(door)
}

func isVillagerWoodenDoor(block coreworld.Block) bool {
	name := block.ResourceLocation()
	return strings.HasSuffix(name, "_door") && !strings.HasSuffix(name, "_trapdoor") && name != "minecraft:iron_door"
}

func (s *Server) setVillagerDoorOpen(position spatial.BlockPos, open bool) bool {
	x, y, z := int(position.X), int(position.Y), int(position.Z)
	lower := s.world.GetBlock(x, y, z)
	want := strconv.FormatBool(open)
	if !isVillagerWoodenDoor(lower) || lower.Properties["half"] == "upper" || lower.Properties["open"] == want {
		return false
	}
	for _, blockY := range []int{y, y + 1} {
		current := s.world.GetBlock(x, blockY, z)
		if current.ResourceLocation() != lower.ResourceLocation() {
			continue
		}
		current.Properties = copyStringMap(current.Properties)
		current.Properties["open"] = want
		s.world.SetBlock(x, blockY, z, current)
		handler.BroadcastBlockChange(coreworld.BlockChange{X: x, Y: blockY, Z: z, Block: current}, s.javaSessionsForDimension(s.simulationDimension))
	}
	return true
}

func (s *Server) villagerNearDoor(position spatial.BlockPos, radius float64) bool {
	x, z := float64(position.X)+0.5, float64(position.Z)+0.5
	for _, entity := range s.world.Entities.Snapshot() {
		if entity.Type == corentity.TypeVillager && !entity.Dead && !entity.Sleeping &&
			math.Hypot(entity.Position.X-x, entity.Position.Z-z) <= radius {
			return true
		}
	}
	return false
}
