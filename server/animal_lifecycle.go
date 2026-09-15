package server

import (
	"math"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
)

// trySheepEatGrass mirrors EatBlockGoal: a sheared sheep occasionally eats a
// grass block below it (turning it to dirt) or short grass/fern at its feet,
// which is what regrows its wool. Returns true when it ate this tick.
func (s *Server) trySheepEatGrass(e *corentity.Entity) bool {
	if s.world == nil {
		return false
	}
	ai := s.mobAIFor(e)
	if ai.rng.Intn(50) != 0 {
		return false
	}
	x := int(math.Floor(e.Position.X))
	y := int(math.Floor(e.Position.Y))
	z := int(math.Floor(e.Position.Z))
	switch s.world.GetBlock(x, y, z).ResourceLocation() {
	case "minecraft:short_grass", "minecraft:grass", "minecraft:fern", "minecraft:tall_grass", "minecraft:large_fern":
		s.setSimulationBlock(x, y, z, coreworld.Air)
		return true
	}
	if s.world.GetBlock(x, y-1, z).ResourceLocation() == "minecraft:grass_block" {
		s.setSimulationBlock(x, y-1, z, coreworld.Block{Namespace: "minecraft", Name: "dirt"})
		return true
	}
	return false
}

// setSimulationBlock writes a block in the simulation dimension and tells clients.
func (s *Server) setSimulationBlock(x, y, z int, block coreworld.Block) {
	s.world.SetBlock(x, y, z, block)
	handler.BroadcastBlockChange(coreworld.BlockChange{X: x, Y: y, Z: z, Block: block},
		s.javaSessionsForDimension(s.simulationDimension))
}

// nextEggLayTicks re-rolls the vanilla chicken EggLayTime (6000-12000 ticks).
func nextEggLayTicks(ai *mobAI) int32 {
	if ai == nil {
		return 6000
	}
	return int32(6000 + ai.rng.Intn(6000))
}

// tickAnimalLifecycle ports Pumpkin's Ageable/Animal/BreedGoal timing to the
// canonical simulation. It is called before passive AI so an assigned mate is
// the entity's movement goal during this tick.
func (s *Server) tickAnimalLifecycle(entities []*corentity.Entity) {
	for _, e := range entities {
		if e == nil || e.Dead {
			continue
		}
		if e.HasTameOwner && s.game != nil {
			if owner := s.game.GetPlayer(e.TameOwnerUUID); owner != nil {
				e.TameOwnerEntityID = owner.EntityID
			} else {
				e.TameOwnerEntityID = 0
			}
		}
		if e.PoisonTicks > 0 {
			e.PoisonTicks--
			if e.PoisonTicks == 0 {
				s.damageEnvironmentalEntity(e, e.MaxHealth, "poison")
			}
		}
		// Sheep wool regrowth. Vanilla regrows wool only by eating a grass block
		// (EatBlockGoal); the timer stays as a fallback so a sheared sheep with no
		// grass nearby still eventually regrows.
		if e.Type == corentity.TypeSheep && e.Sheared {
			if s.trySheepEatGrass(e) {
				e.Sheared = false
				e.WoolRegrowTicks = 0
				s.broadcastAnimalEvent(e, 10, 0) // Java EATING_GRASS entity status
				handler.BroadcastMobMetadata(e, s.sessions)
			} else if e.WoolRegrowTicks > 0 {
				e.WoolRegrowTicks--
				if e.WoolRegrowTicks == 0 {
					e.Sheared = false
					handler.BroadcastMobMetadata(e, s.sessions)
				}
			}
		}
		// Chicken egg laying: adult, non-jockey chickens drop an egg every
		// 6000-12000 ticks (vanilla EggLayTime), re-rolling the timer each lay.
		if e.Type == corentity.TypeChicken && !e.IsBaby && e.RiderEntityID == 0 {
			if e.EggLayTicks <= 0 {
				e.EggLayTicks = nextEggLayTicks(s.mobAIFor(e))
			}
			e.EggLayTicks--
			if e.EggLayTicks <= 0 {
				if s.world != nil {
					s.newDroppedItemInWorld(s.world, player.ItemStack{ItemID: "minecraft:egg", Count: 1}, e.Position, int(e.EntityID))
				}
				handler.BroadcastSoundAt(s.sessions, "minecraft:entity.chicken.egg", handler.SoundCategoryNeutral,
					e.Position.X, e.Position.Y, e.Position.Z, 1, 1)
				e.EggLayTicks = nextEggLayTicks(s.mobAIFor(e))
			}
		}
		if !corentity.IsAgeableAnimal(e.Type) {
			continue
		}
		if e.IsBaby {
			e.BabyAgeTicks++
			if e.BabyAgeTicks >= corentity.BabyGrowUpTicks {
				e.IsBaby = false
				e.BabyAgeTicks = 0
				handler.BroadcastMobMetadata(e, s.sessions)
			}
		}
		if e.BreedingCooldownTicks > 0 {
			e.BreedingCooldownTicks--
		}
		if e.LoveTicks > 0 {
			e.LoveTicks--
			if e.LoveTicks == 0 {
				clearBreedingPair(e)
			}
		}
	}

	for _, e := range entities {
		if !animalReadyToMate(e) || e.BreedingMateEntityID != 0 {
			continue
		}
		if ai := s.mobAIs[e.EntityID]; ai != nil && ai.panicTick > 0 {
			continue
		}
		mate := s.closestCompatibleMate(e, entities)
		if mate == nil {
			continue
		}
		e.BreedingMateEntityID = mate.EntityID
		e.BreedingProgressTicks = 0
		mate.BreedingMateEntityID = e.EntityID
		mate.BreedingProgressTicks = 0
	}

	for _, e := range entities {
		if !animalReadyToMate(e) || e.BreedingMateEntityID == 0 {
			continue
		}
		mate, ok := s.world.Entities.Get(e.BreedingMateEntityID)
		if !ok || !animalReadyToMate(mate) || mate.BreedingMateEntityID != e.EntityID ||
			!corentity.CompatibleBreedingTypes(e.Type, mate.Type) {
			clearBreedingPair(e)
			continue
		}
		if (s.mobAIs[e.EntityID] != nil && s.mobAIs[e.EntityID].panicTick > 0) ||
			(s.mobAIs[mate.EntityID] != nil && s.mobAIs[mate.EntityID].panicTick > 0) {
			clearBreedingPair(e)
			clearBreedingPair(mate)
			continue
		}
		// The lower ID owns pair progress and birth, preventing double offspring.
		if e.EntityID > mate.EntityID {
			continue
		}
		dx := e.Position.X - mate.Position.X
		dy := e.Position.Y - mate.Position.Y
		dz := e.Position.Z - mate.Position.Z
		distanceSquared := dx*dx + dy*dy + dz*dz
		if distanceSquared > 8*8 {
			clearBreedingPair(e)
			clearBreedingPair(mate)
			continue
		}
		e.BreedingProgressTicks++
		mate.BreedingProgressTicks = e.BreedingProgressTicks
		if e.BreedingProgressTicks < corentity.BreedingDelayTicks || distanceSquared >= 3*3 {
			continue
		}
		s.spawnAnimalChild(e, mate)
	}
}

func animalReadyToMate(e *corentity.Entity) bool {
	return e != nil && !e.Dead && !e.IsBaby && corentity.IsBreedableAnimal(e.Type) &&
		e.LoveTicks > 0 && e.BreedingCooldownTicks == 0
}

func clearBreedingPair(e *corentity.Entity) {
	if e == nil {
		return
	}
	e.BreedingMateEntityID = 0
	e.BreedingProgressTicks = 0
}

func (s *Server) closestCompatibleMate(e *corentity.Entity, entities []*corentity.Entity) *corentity.Entity {
	var closest *corentity.Entity
	closestDistance := float64(8 * 8)
	for _, candidate := range entities {
		if candidate == e || !animalReadyToMate(candidate) || candidate.BreedingMateEntityID != 0 ||
			!corentity.CompatibleBreedingTypes(e.Type, candidate.Type) {
			continue
		}
		if ai := s.mobAIs[candidate.EntityID]; ai != nil && ai.panicTick > 0 {
			continue
		}
		dx := e.Position.X - candidate.Position.X
		dy := e.Position.Y - candidate.Position.Y
		dz := e.Position.Z - candidate.Position.Z
		distance := dx*dx + dy*dy + dz*dz
		if distance <= closestDistance {
			closest = candidate
			closestDistance = distance
		}
	}
	return closest
}

func (s *Server) spawnAnimalChild(first, second *corentity.Entity) *corentity.Entity {
	childType := corentity.BreedingChildType(first.Type, second.Type)
	positionX := (first.Position.X + second.Position.X) / 2
	positionY := math.Max(first.Position.Y, second.Position.Y)
	positionZ := (first.Position.Z + second.Position.Z) / 2
	child := corentity.New(s.game.NextEntityID(), newRandomUUID(), childType, positionX, positionY, positionZ)
	child.IsBaby = true
	child.BabyAgeTicks = 0
	child.OnGround = first.OnGround || second.OnGround
	if (childType == corentity.TypeWolf || childType == corentity.TypeCat) && (first.Tamed || second.Tamed) {
		owner := first
		if !owner.HasTameOwner {
			owner = second
		}
		child.Tamed = true
		child.HasTameOwner = owner.HasTameOwner
		child.TameOwnerUUID = owner.TameOwnerUUID
		child.TameOwnerEntityID = owner.TameOwnerEntityID
	}
	s.world.Entities.Add(child)
	handler.BroadcastSpawnMob(child, s.sessions)

	for _, parent := range []*corentity.Entity{first, second} {
		parent.LoveTicks = 0
		parent.HasLoveCause = false
		parent.BreedingCooldownTicks = corentity.BreedingCooldownTicks
		clearBreedingPair(parent)
		handler.BroadcastMobMetadata(parent, s.sessions)
	}
	s.broadcastAnimalEvent(first, 18, 26)
	s.broadcastAnimalEvent(second, 18, 26)
	return child
}
