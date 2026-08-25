package server

import (
	"math"
	"math/rand"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

// EndermanTeleportRadius is the max block radius for a damage-triggered teleport.
const EndermanTeleportRadius = 32

// tryEndermanTeleport mirrors Pumpkin's 64 random landing attempts. It only
// considers loaded chunks and rejects water, blocked headroom, and void floors.
func (s *Server) tryEndermanTeleport(enderman *corentity.Entity) bool {
	if s == nil || s.world == nil || enderman == nil {
		return false
	}
	seed := int64(enderman.EntityID)*6364136223846793005 + enderman.AgeTicks
	rng := rand.New(rand.NewSource(seed))
	for attempt := 0; attempt < 64; attempt++ {
		x := enderman.Position.X + (rng.Float64()-0.5)*64
		z := enderman.Position.Z + (rng.Float64()-0.5)*64
		feetY := int(math.Floor(enderman.Position.Y)) + rng.Intn(64) - 32
		cx, cz := coreworld.ChunkCoordsFor(int(math.Floor(x)), int(math.Floor(z)))
		if !s.world.IsChunkLoaded(cx, cz) {
			continue
		}
		for feetY > coreworld.WorldMinY+1 &&
			!coreworld.IsEntitySupportBlock(s.world.GetBlock(int(math.Floor(x)), feetY-1, int(math.Floor(z))).ResourceLocation()) {
			feetY--
		}
		if feetY <= coreworld.WorldMinY+1 || s.world.TouchesWater(x, float64(feetY), z) {
			continue
		}
		ok, loaded := s.world.CanEntityOccupyIfLoaded(x, float64(feetY), z)
		if !loaded || !ok || coreworld.IsEntitySupportBlock(
			s.world.GetBlock(int(math.Floor(x)), feetY+2, int(math.Floor(z))).ResourceLocation()) {
			continue
		}
		enderman.Position.X, enderman.Position.Y, enderman.Position.Z = x, float64(feetY), z
		enderman.VX, enderman.VY, enderman.VZ = 0, 0, 0
		return true
	}
	return false
}

// EndermanPickupBlocks lists block IDs an enderman can pick up.
// TODO: wire into mob_actions block-pickup event.
var EndermanPickupBlocks = map[string]bool{
	"minecraft:grass_block":    true,
	"minecraft:dirt":           true,
	"minecraft:sand":           true,
	"minecraft:gravel":         true,
	"minecraft:brown_mushroom": true,
	"minecraft:red_mushroom":   true,
	"minecraft:cactus":         true,
	"minecraft:pumpkin":        true,
	"minecraft:melon":          true,
	"minecraft:mycelium":       true,
}
