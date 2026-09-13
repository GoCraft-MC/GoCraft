package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

// zombieDoorBreakTicks mirrors vanilla BreakDoorGoal.DEFAULT_DOOR_BREAK_TIME.
const zombieDoorBreakTicks = 240

func zombieCanBreakDoors(t corentity.EntityType) bool {
	switch t {
	case corentity.TypeZombie, corentity.TypeHusk, corentity.TypeZombieVillager, corentity.TypeDrowned:
		return true
	default:
		return false
	}
}

// tickZombieDoorBreak ports vanilla BreakDoorGoal: on hard difficulty a zombie
// stuck against a closed wooden door on the way to its target bashes it down
// over 240 ticks. GoCraft zombies path with doors treated as solid, so a zombie
// hunting a target behind a door ends up adjacent to it — the trigger here.
// Returns true while the zombie is actively breaking (owning the tick).
func (s *Server) tickZombieDoorBreak(e *corentity.Entity, ai *mobAI) bool {
	if e == nil || ai == nil || s.world == nil {
		return false
	}
	if !zombieCanBreakDoors(e.Type) || s.cfg == nil || s.cfg.Difficulty != "hard" {
		ai.doorBreakTick = 0
		ai.doorBreakPos = spatial.BlockPos{}
		return false
	}
	doorPos, found := s.nearestClosedWoodenDoor(e)
	if !found {
		ai.doorBreakTick = 0
		ai.doorBreakPos = spatial.BlockPos{}
		return false
	}
	if ai.doorBreakPos != doorPos {
		ai.doorBreakPos = doorPos
		ai.doorBreakTick = 0
	}
	e.VX, e.VZ = 0, 0
	ai.doorBreakTick++
	if ai.doorBreakTick%20 == 0 {
		handler.BroadcastSoundAt(s.sessions, "minecraft:entity.zombie.attack_wooden_door", handler.SoundCategoryHostile,
			float64(doorPos.X)+0.5, float64(doorPos.Y)+0.5, float64(doorPos.Z)+0.5, 1, 1)
	}
	if ai.doorBreakTick >= zombieDoorBreakTicks {
		s.breakWoodenDoor(doorPos)
		ai.doorBreakTick = 0
		ai.doorBreakPos = spatial.BlockPos{}
	}
	return true
}

func (s *Server) nearestClosedWoodenDoor(e *corentity.Entity) (spatial.BlockPos, bool) {
	zx := int(math.Floor(e.Position.X))
	zy := int(math.Floor(e.Position.Y))
	zz := int(math.Floor(e.Position.Z))
	for dx := -1; dx <= 1; dx++ {
		for dz := -1; dz <= 1; dz++ {
			if dx == 0 && dz == 0 {
				continue
			}
			pos, door, ok := s.villagerDoorAt(zx+dx, zy, zz+dz)
			if ok && door.Properties["open"] != "true" {
				return pos, true
			}
		}
	}
	return spatial.BlockPos{}, false
}

func (s *Server) breakWoodenDoor(pos spatial.BlockPos) {
	x, y, z := int(pos.X), int(pos.Y), int(pos.Z)
	lower := s.world.GetBlock(x, y, z)
	if !isVillagerWoodenDoor(lower) {
		return
	}
	for _, blockY := range []int{y, y + 1} {
		current := s.world.GetBlock(x, blockY, z)
		if current.ResourceLocation() != lower.ResourceLocation() {
			continue
		}
		s.world.SetBlock(x, blockY, z, coreworld.Air)
		handler.BroadcastBlockChange(coreworld.BlockChange{X: x, Y: blockY, Z: z, Block: coreworld.Air},
			s.javaSessionsForDimension(s.simulationDimension))
	}
	handler.BroadcastSoundAt(s.sessions, "minecraft:entity.zombie.break_wooden_door", handler.SoundCategoryHostile,
		float64(x)+0.5, float64(y)+0.5, float64(z)+0.5, 1, 1)
}
