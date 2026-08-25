package world

import (
	"math"
	"strings"
	"sync"
)

const strongholdCount = 128

var strongholdPlacementCache sync.Map

func (r *legacyRandom) nextLong() int64 {
	high := int64(int32(r.next(32)))
	low := int64(int32(r.next(32)))
	return (high << 32) + low
}

// strongholdRingChunks mirrors Pumpkin's concentric-rings placement. Biome
// relocation is applied separately so the main random stream stays unchanged.
func (g *OverworldGenerator) strongholdRingChunks() [][2]int {
	if cached, ok := strongholdPlacementCache.Load(g.seed); ok {
		return cached.([][2]int)
	}
	rng := newLegacyRandom(g.seed)
	angle := rng.nextDouble() * math.Pi * 2
	spread, ring, inRing := 3, 0, 0
	chunks := make([][2]int, 0, strongholdCount)
	for index := 0; index < strongholdCount; index++ {
		distance := 128.0 + float64(ring*192) + (rng.nextDouble()-0.5)*80
		chunkX := int(math.Floor(math.Cos(angle)*distance + 0.5))
		chunkZ := int(math.Floor(math.Sin(angle)*distance + 0.5))
		forkSeed := rng.nextLong()
		chunkX, chunkZ = g.relocateStronghold(chunkX, chunkZ, forkSeed)
		chunks = append(chunks, [2]int{chunkX, chunkZ})
		angle += math.Pi * 2 / float64(spread)
		inRing++
		if inRing == spread {
			ring++
			inRing = 0
			spread += 2 * spread / (ring + 1)
			spread = min(spread, strongholdCount-index)
			angle += rng.nextDouble() * math.Pi * 2
		}
	}
	strongholdPlacementCache.Store(g.seed, chunks)
	return chunks
}

func (g *OverworldGenerator) relocateStronghold(chunkX, chunkZ int, seed int64) (int, int) {
	rng := newLegacyRandom(seed)
	centerX, centerZ := chunkX*16+8, chunkZ*16+8
	foundX, foundZ, foundCount := chunkX, chunkZ, 0
	for dz := -112; dz <= 112; dz += 4 {
		for dx := -112; dx <= 112; dx += 4 {
			testX, testZ := centerX+dx, centerZ+dz
			if !strongholdBiome(g.BiomeAt3D(testX, 0, testZ)) {
				continue
			}
			foundCount++
			if foundCount == 1 || rng.nextInt(foundCount) == 0 {
				foundX, foundZ = testX>>4, testZ>>4
			}
		}
	}
	return foundX, foundZ
}

func strongholdBiome(name string) bool {
	switch strings.TrimPrefix(name, "minecraft:") {
	case "beach", "cold_ocean", "deep_cold_ocean", "deep_dark",
		"deep_frozen_ocean", "deep_lukewarm_ocean", "deep_ocean",
		"frozen_ocean", "frozen_river", "lukewarm_ocean",
		"mangrove_swamp", "ocean", "river", "snowy_beach",
		"stony_shore", "swamp", "warm_ocean":
		return false
	default:
		return true
	}
}

// NearestStronghold returns the centre block of the closest ring position.
// Pumpkin resolves concentric placements in one pass, independent of the
// random-spread search radius supplied by the Eye-of-Ender item.
func (g *OverworldGenerator) NearestStronghold(x, z, _ int) (int, int, bool) {
	bestDistance := math.MaxFloat64
	bestX, bestZ, found := 0, 0, false
	for _, chunk := range g.strongholdRingChunks() {
		blockX, blockZ := chunk[0]*16+8, chunk[1]*16+8
		dx, dz := float64(blockX-x), float64(blockZ-z)
		if distance := dx*dx + dz*dz; distance <= bestDistance {
			bestDistance, bestX, bestZ, found = distance, blockX, blockZ, true
		}
	}
	return bestX, bestZ, found
}
