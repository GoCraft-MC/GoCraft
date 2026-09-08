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

const villagerDoorCloseTicks = 60

// tickVillagerDoor mirrors the wooden-door capability used by Pumpkin's walk
// node evaluator. It runs on the serial tick phase because villagers may share
// an entrance while their ordinary AI is evaluated in parallel.
func (s *Server) tickVillagerDoor(villager *corentity.Entity, ai *mobAI) {
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
	if ai.pathIndex >= len(ai.path) {
		return
	}
	waypoint := ai.path[ai.pathIndex]
	position, door, ok := s.villagerDoorAt(int(math.Floor(waypoint.X)), int(math.Floor(waypoint.Y)), int(math.Floor(waypoint.Z)))
	if !ok || door.Properties["open"] == "true" {
		return
	}
	dx, dz := waypoint.X-villager.Position.X, waypoint.Z-villager.Position.Z
	if dx*dx+dz*dz > 1.75*1.75 {
		return
	}
	if s.setVillagerDoorOpen(position, true) {
		ai.openedDoor = position
		ai.doorCloseTick = villagerDoorCloseTicks
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
