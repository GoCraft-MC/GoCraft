package world

import (
	"math"
	"sort"
)

const (
	endOuterIslandStart = 900.0
	endIslandCellSize   = 192
)

var (
	endStoneBlock    = block("end_stone")
	endPortalBlock   = block("end_portal")
	obsidianBlock    = block("obsidian")
	ironBarsBlock    = block("iron_bars")
	chorusPlantBlock = block("chorus_plant")
	chorusFlower     = blockProps("chorus_flower", "age", "0")
)

// EndGenerator implements the recognizable vanilla/Pumpkin End topology:
// the central dragon island and spikes, the arrival and return portals, a wide
// inner void, then deterministic large and small outer islands with End biome
// bands and chorus vegetation.
type EndGenerator struct{ seed int64 }

func NewEndGenerator(seed int64) *EndGenerator { return &EndGenerator{seed: seed} }
func (g *EndGenerator) Seed() int64            { return g.seed }

func (g *EndGenerator) Generate(chunkX, chunkZ int32) *Chunk {
	c := &Chunk{X: chunkX, Z: chunkZ}
	minX, minZ := int(chunkX)*SectionSize, int(chunkZ)*SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		worldX := minX + localX
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldZ := minZ + localZ
			surface, bottom, exists, _ := g.endTerrainAt(worldX, worldZ)
			if !exists {
				continue
			}
			for y := bottom; y <= surface; y++ {
				setGeneratedBlock(c, localX, y, localZ, endStoneBlock)
			}
		}
	}
	g.addEndSpikes(c)
	g.addEndArrivalPlatform(c)
	g.addEndExitPortal(c)
	g.addChorusPlants(c)
	g.populateEndBiomes(c)
	return c
}

// BiomeAt returns the surface End biome at an absolute column.
func (g *EndGenerator) BiomeAt(x, z int) string { return g.BiomeAt3D(x, 64, z) }

func (g *EndGenerator) BiomeAt3D(x, _ int, z int) string {
	distance := math.Hypot(float64(x), float64(z))
	if distance < endOuterIslandStart {
		return "minecraft:the_end"
	}
	_, _, exists, influence := g.endTerrainAt(x, z)
	if !exists || influence < 0.20 {
		return "minecraft:small_end_islands"
	}
	switch {
	case influence >= 0.67:
		return "minecraft:end_highlands"
	case influence >= 0.43:
		return "minecraft:end_midlands"
	default:
		return "minecraft:end_barrens"
	}
}

// endTerrainAt returns a solid vertical interval and the outer-island
// influence. The radial central island mirrors the vanilla End's falloff;
// outer islands use seeded cells, which preserve the large gaps and clustered
// land masses of Pumpkin's End density router.
func (g *EndGenerator) endTerrainAt(x, z int) (surface, bottom int, exists bool, influence float64) {
	distance := math.Hypot(float64(x), float64(z))
	centralEdge := 82.0 + dimensionFractal2D(g.seed, float64(x), float64(z), 34, 3, 0x63656e7472616c)*14
	if distance <= centralEdge {
		ratio := distance / centralEdge
		topNoise := dimensionFractal2D(g.seed, float64(x), float64(z), 23, 2, 0x656e64746f70)
		surface = 64 - int(math.Pow(ratio, 1.65)*25) + int(math.Round(topNoise*3))
		depth := 12 + int((1-ratio)*23)
		return surface, surface - depth, true, 1 - ratio
	}
	if distance < endOuterIslandStart {
		return 0, 0, false, 0
	}

	cellX, cellZ := floorDiv(x, endIslandCellSize), floorDiv(z, endIslandCellSize)
	best := -1.0
	for candidateX := cellX - 1; candidateX <= cellX+1; candidateX++ {
		for candidateZ := cellZ - 1; candidateZ <= cellZ+1; candidateZ++ {
			hash := generatedHash(g.seed^0x6f75746572656e64, candidateX, 0, candidateZ)
			if hash%100 >= 68 {
				continue
			}
			centerX := candidateX*endIslandCellSize + 24 + int((hash>>8)%144)
			centerZ := candidateZ*endIslandCellSize + 24 + int((hash>>20)%144)
			if math.Hypot(float64(centerX), float64(centerZ)) < endOuterIslandStart {
				continue
			}
			radius := 48.0 + float64((hash>>32)%72)
			d := math.Hypot(float64(x-centerX), float64(z-centerZ))
			candidate := 1 - d/radius
			if candidate > best {
				best = candidate
			}
		}
	}
	if best <= 0 {
		return 0, 0, false, 0
	}
	topNoise := dimensionFractal2D(g.seed, float64(x), float64(z), 27, 3, 0x6f75746572746f70)
	surface = 54 + int(best*28) + int(math.Round(topNoise*4))
	depth := 5 + int(best*24)
	return surface, surface - depth, true, best
}

type endSpike struct {
	x, z    int
	radius  int
	height  int
	guarded bool
}

func (g *EndGenerator) endSpikes() []endSpike {
	sizes := make([]int, 10)
	for i := range sizes {
		sizes[i] = i
	}
	state := generatedHash(g.seed^0x7370696b6573, 0, 0, 0)
	for i := len(sizes) - 1; i > 0; i-- {
		state = mix64(state)
		j := int(state % uint64(i+1))
		sizes[i], sizes[j] = sizes[j], sizes[i]
	}
	spikes := make([]endSpike, 0, 10)
	for i, size := range sizes {
		angle := float64(i) * math.Pi * 2 / 10
		spikes = append(spikes, endSpike{
			x:       int(math.Floor(42 * math.Cos(angle))),
			z:       int(math.Floor(42 * math.Sin(angle))),
			radius:  2 + size/3,
			height:  76 + size*3,
			guarded: size == 1 || size == 2,
		})
	}
	sort.Slice(spikes, func(i, j int) bool {
		if spikes[i].x == spikes[j].x {
			return spikes[i].z < spikes[j].z
		}
		return spikes[i].x < spikes[j].x
	})
	return spikes
}

func (g *EndGenerator) addEndSpikes(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	maxX, maxZ := minX+SectionSize-1, minZ+SectionSize-1
	for _, spike := range g.endSpikes() {
		if spike.x+spike.radius < minX || spike.x-spike.radius > maxX || spike.z+spike.radius < minZ || spike.z-spike.radius > maxZ {
			continue
		}
		for worldX := max(minX, spike.x-spike.radius); worldX <= min(maxX, spike.x+spike.radius); worldX++ {
			for worldZ := max(minZ, spike.z-spike.radius); worldZ <= min(maxZ, spike.z+spike.radius); worldZ++ {
				dx, dz := worldX-spike.x, worldZ-spike.z
				if dx*dx+dz*dz > spike.radius*spike.radius+1 {
					continue
				}
				for y := 0; y < spike.height; y++ {
					setGeneratedBlock(c, worldX-minX, y, worldZ-minZ, obsidianBlock)
				}
			}
		}
		if spike.x < minX || spike.x > maxX || spike.z < minZ || spike.z > maxZ {
			continue
		}
		localX, localZ := spike.x-minX, spike.z-minZ
		setGeneratedBlock(c, localX, spike.height, localZ, bedrockBlock)
		setGeneratedBlock(c, localX, spike.height+1, localZ, fireBlock)
		if spike.guarded {
			for dy := 0; dy <= 3; dy++ {
				for dx := -2; dx <= 2; dx++ {
					for dz := -2; dz <= 2; dz++ {
						if dx != -2 && dx != 2 && dz != -2 && dz != 2 && dy != 3 {
							continue
						}
						px, pz := localX+dx, localZ+dz
						if px >= 0 && px < SectionSize && pz >= 0 && pz < SectionSize {
							setGeneratedBlock(c, px, spike.height+dy, pz, ironBarsBlock)
						}
					}
				}
			}
		}
	}
}

func (g *EndGenerator) addEndArrivalPlatform(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for worldX := 98; worldX <= 102; worldX++ {
		for worldZ := -2; worldZ <= 2; worldZ++ {
			if worldX < minX || worldX >= minX+SectionSize || worldZ < minZ || worldZ >= minZ+SectionSize {
				continue
			}
			localX, localZ := worldX-minX, worldZ-minZ
			setGeneratedBlock(c, localX, 48, localZ, obsidianBlock)
			for y := 49; y <= 51; y++ {
				setGeneratedBlock(c, localX, y, localZ, Air)
			}
		}
	}
}

func (g *EndGenerator) addEndExitPortal(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for worldX := -2; worldX <= 2; worldX++ {
		for worldZ := -2; worldZ <= 2; worldZ++ {
			if worldX < minX || worldX >= minX+SectionSize || worldZ < minZ || worldZ >= minZ+SectionSize {
				continue
			}
			localX, localZ := worldX-minX, worldZ-minZ
			setGeneratedBlock(c, localX, 63, localZ, bedrockBlock)
			if absInt(worldX) <= 1 && absInt(worldZ) <= 1 {
				setGeneratedBlock(c, localX, 64, localZ, endPortalBlock)
			} else {
				setGeneratedBlock(c, localX, 64, localZ, bedrockBlock)
			}
		}
	}
	if minX <= 0 && minX+SectionSize > 0 && minZ <= 0 && minZ+SectionSize > 0 {
		localX, localZ := -minX, -minZ
		for y := 65; y <= 68; y++ {
			setGeneratedBlock(c, localX, y, localZ, bedrockBlock)
		}
	}
}

func (g *EndGenerator) addChorusPlants(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for localX := 1; localX < SectionSize-1; localX++ {
		worldX := minX + localX
		for localZ := 1; localZ < SectionSize-1; localZ++ {
			worldZ := minZ + localZ
			biome := g.BiomeAt(worldX, worldZ)
			if biome != "minecraft:end_highlands" && biome != "minecraft:end_midlands" {
				continue
			}
			top := c.HighestBlockY(localX, localZ)
			if top < 1 || generatedBlock(c, localX, top, localZ).ResourceLocation() != "minecraft:end_stone" {
				continue
			}
			state := generatedHash(g.seed^0x63686f727573, worldX, top, worldZ)
			if state%47 != 0 {
				continue
			}
			height := 3 + int((state>>10)%5)
			for offset := 1; offset <= height; offset++ {
				setGeneratedBlock(c, localX, top+offset, localZ, chorusPlantBlock)
			}
			setGeneratedBlock(c, localX, top+height, localZ, chorusFlower)
			if height >= 5 {
				direction := int((state >> 20) % 4)
				dx, dz := [4]int{1, -1, 0, 0}[direction], [4]int{0, 0, 1, -1}[direction]
				branchY := top + height - 2
				setGeneratedBlock(c, localX+dx, branchY, localZ+dz, chorusPlantBlock)
				setGeneratedBlock(c, localX+dx, branchY+1, localZ+dz, chorusFlower)
			}
		}
	}
}

func (g *EndGenerator) populateEndBiomes(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for sectionIndex, section := range c.Sections {
		if section == nil {
			continue
		}
		sectionY := SectionMinY(sectionIndex)
		section.SetUniformBiome(g.BiomeAt3D(minX+2, sectionY+2, minZ+2))
		for quartY := 0; quartY < 4; quartY++ {
			for quartZ := 0; quartZ < 4; quartZ++ {
				for quartX := 0; quartX < 4; quartX++ {
					section.SetBiomeCell(quartX, quartY, quartZ, g.BiomeAt3D(minX+quartX*4+2, sectionY+quartY*4+2, minZ+quartZ*4+2))
				}
			}
		}
	}
}
