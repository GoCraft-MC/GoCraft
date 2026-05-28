package world

import "math"

const (
	netherMinY     = 0
	netherMaxY     = 127
	netherLavaY    = 31
	netherBiomeY   = 64
	netherNoiseKey = int64(0x4e6574686572)
)

var (
	netherrackBlock      = block("netherrack")
	netherQuartzOreBlock = block("nether_quartz_ore")
	netherGoldOreBlock   = block("nether_gold_ore")
	ancientDebrisBlock   = block("ancient_debris")
	blackstoneBlock      = block("blackstone")
	basaltBlock          = block("basalt")
	magmaBlock           = block("magma_block")
	soulSandBlock        = block("soul_sand")
	soulSoilBlock        = block("soul_soil")
	crimsonNyliumBlock   = block("crimson_nylium")
	warpedNyliumBlock    = block("warped_nylium")
	crimsonStemBlock     = block("crimson_stem")
	warpedStemBlock      = block("warped_stem")
	netherWartBlock      = block("nether_wart_block")
	warpedWartBlock      = block("warped_wart_block")
	shroomlightBlock     = block("shroomlight")
	glowstoneBlock       = block("glowstone")
	fireBlock            = block("fire")
	soulFireBlock        = block("soul_fire")
	crimsonRootsBlock    = block("crimson_roots")
	warpedRootsBlock     = block("warped_roots")
	netherSproutsBlock   = block("nether_sprouts")
	crimsonFungusBlock   = block("crimson_fungus")
	warpedFungusBlock    = block("warped_fungus")
	weepingVinesBlock    = block("weeping_vines")
	twistingVinesBlock   = block("twisting_vines")
)

// NetherGenerator ports Pumpkin's dimension-specific generation stages to a
// native Go pipeline: a 3-D density field, lava sea, biome surface rules, ores
// and biome decoration. It preserves the canonical vanilla 0..127 Nether band
// inside GoCraft's shared world-height representation.
type NetherGenerator struct{ seed int64 }

func NewNetherGenerator(seed int64) *NetherGenerator { return &NetherGenerator{seed: seed} }
func (g *NetherGenerator) Seed() int64               { return g.seed }

func (g *NetherGenerator) Generate(chunkX, chunkZ int32) *Chunk {
	c := &Chunk{X: chunkX, Z: chunkZ}
	minX, minZ := int(chunkX)*SectionSize, int(chunkZ)*SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		worldX := minX + localX
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldZ := minZ + localZ
			for y := netherMinY; y <= netherMaxY; y++ {
				material := Air
				switch {
				case g.netherBedrockAt(worldX, y, worldZ):
					material = bedrockBlock
				case g.netherDensity(worldX, y, worldZ) > 0:
					material = netherrackBlock
				case y <= netherLavaY:
					material = lavaBlock
				}
				if !material.IsAir() {
					setGeneratedBlock(c, localX, y, localZ, material)
				}
			}
		}
	}
	g.applyNetherSurfaceRules(c)
	g.addNetherOres(c)
	g.decorateNether(c)
	g.populateNetherBiomes(c)
	return c
}

func (g *NetherGenerator) netherBedrockAt(x, y, z int) bool {
	if y == netherMinY || y == netherMaxY {
		return true
	}
	if y > 4 && y < 123 {
		return false
	}
	distance := y
	salt := int64(0x666c6f6f72)
	if y > 4 {
		distance = netherMaxY - y
		salt = 0x726f6f66
	}
	return int(generatedHash(g.seed^salt, x, y, z)%5) >= distance
}

func (g *NetherGenerator) netherDensity(x, y, z int) float64 {
	largeCaves := dimensionFractal3D(g.seed^netherNoiseKey, float64(x), float64(y), float64(z), 74, 48, 74, 3, 0x64656e7369747931)
	ridges := 1 - math.Abs(dimensionNoise3D(g.seed, float64(x), float64(y), float64(z), 31, 24, 31, 0x7269646765733031))
	ridges = (ridges - 0.62) * 0.72
	lowerShelf := math.Max(0, (37-float64(y))/21)
	upperShelf := math.Max(0, (float64(y)-91)/21)
	return largeCaves*1.05 + ridges + lowerShelf + upperShelf - 0.25
}

// BiomeAt and BiomeAt3D expose the same multi-noise biome selection used when
// writing chunk biome palettes. Natural spawning can therefore use Nether
// biome spawn tables without first loading a chunk.
func (g *NetherGenerator) BiomeAt(x, z int) string { return g.BiomeAt3D(x, netherBiomeY, z) }

func (g *NetherGenerator) BiomeAt3D(x, y, z int) string {
	temperature := dimensionFractal2D(g.seed, float64(x), float64(z), 190, 3, 0x74656d7065726174)
	humidity := dimensionFractal2D(g.seed, float64(x), float64(z), 175, 3, 0x68756d6964697479)
	weirdness := dimensionNoise3D(g.seed, float64(x), float64(y), float64(z), 145, 96, 145, 0x77656972646e6573)
	switch {
	case weirdness > 0.43:
		return "minecraft:basalt_deltas"
	case weirdness < -0.43:
		return "minecraft:soul_sand_valley"
	case temperature > 0.24 || humidity > 0.48:
		return "minecraft:crimson_forest"
	case temperature < -0.24 || humidity < -0.48:
		return "minecraft:warped_forest"
	default:
		return "minecraft:nether_wastes"
	}
}

func (g *NetherGenerator) applyNetherSurfaceRules(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldX, worldZ := minX+localX, minZ+localZ
			for y := 5; y <= 122; y++ {
				if generatedBlock(c, localX, y, localZ).ResourceLocation() != "minecraft:netherrack" {
					continue
				}
				floorSurface := generatedBlock(c, localX, y+1, localZ).IsAir()
				ceilingSurface := generatedBlock(c, localX, y-1, localZ).IsAir()
				if !floorSurface && !ceilingSurface {
					continue
				}
				biome := g.BiomeAt3D(worldX, y, worldZ)
				material := netherrackBlock
				switch biome {
				case "minecraft:crimson_forest":
					if floorSurface {
						material = crimsonNyliumBlock
					}
				case "minecraft:warped_forest":
					if floorSurface {
						material = warpedNyliumBlock
					}
				case "minecraft:soul_sand_valley":
					if generatedHash(g.seed, worldX, y, worldZ)%4 == 0 {
						material = soulSoilBlock
					} else {
						material = soulSandBlock
					}
				case "minecraft:basalt_deltas":
					if generatedHash(g.seed, worldX, y, worldZ)%5 == 0 {
						material = blackstoneBlock
					} else {
						material = basaltBlock
					}
				}
				setGeneratedBlock(c, localX, y, localZ, material)
			}
		}
	}
}

func (g *NetherGenerator) addNetherOres(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		worldX := minX + localX
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldZ := minZ + localZ
			for y := 6; y <= 121; y++ {
				if generatedBlock(c, localX, y, localZ).ResourceLocation() != "minecraft:netherrack" {
					continue
				}
				hash := generatedHash(g.seed^0x6e65746865726f72, worldX, y, worldZ)
				quartz := dimensionNoise3D(g.seed, float64(worldX), float64(y), float64(worldZ), 4.6, 4.2, 4.6, 0x71756172747a)
				gold := dimensionNoise3D(g.seed, float64(worldX), float64(y), float64(worldZ), 3.8, 4.8, 3.8, 0x676f6c64)
				switch {
				case y >= 8 && y <= 22 && hash%2600 == 0:
					setGeneratedBlock(c, localX, y, localZ, ancientDebrisBlock)
				case quartz > 0.58 && hash%4 == 0:
					setGeneratedBlock(c, localX, y, localZ, netherQuartzOreBlock)
				case gold > 0.67 && hash%5 == 0:
					setGeneratedBlock(c, localX, y, localZ, netherGoldOreBlock)
				}
			}
		}
	}
}

func (g *NetherGenerator) decorateNether(c *Chunk) {
	minX, minZ := int(c.X)*SectionSize, int(c.Z)*SectionSize
	for localX := 0; localX < SectionSize; localX++ {
		worldX := minX + localX
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldZ := minZ + localZ
			for y := 32; y <= 121; y++ {
				if !generatedBlock(c, localX, y, localZ).IsAir() {
					continue
				}
				hash := generatedHash(g.seed^0x6665617475726573, worldX, y, worldZ)
				below := generatedBlock(c, localX, y-1, localZ).ResourceLocation()
				if below != "minecraft:air" && below != "minecraft:lava" {
					biome := g.BiomeAt3D(worldX, y, worldZ)
					switch biome {
					case "minecraft:crimson_forest":
						if hash%521 == 0 && localX >= 2 && localX <= 13 && localZ >= 2 && localZ <= 13 {
							g.placeHugeNetherFungus(c, localX, y, localZ, false, hash)
						} else if hash%17 == 0 {
							setGeneratedBlock(c, localX, y, localZ, crimsonRootsBlock)
						} else if hash%47 == 0 {
							setGeneratedBlock(c, localX, y, localZ, crimsonFungusBlock)
						}
					case "minecraft:warped_forest":
						if hash%521 == 0 && localX >= 2 && localX <= 13 && localZ >= 2 && localZ <= 13 {
							g.placeHugeNetherFungus(c, localX, y, localZ, true, hash)
						} else if hash%13 == 0 {
							setGeneratedBlock(c, localX, y, localZ, netherSproutsBlock)
						} else if hash%19 == 0 {
							setGeneratedBlock(c, localX, y, localZ, warpedRootsBlock)
						} else if hash%53 == 0 {
							setGeneratedBlock(c, localX, y, localZ, warpedFungusBlock)
						}
					case "minecraft:soul_sand_valley":
						if hash%31 == 0 {
							setGeneratedBlock(c, localX, y, localZ, soulFireBlock)
						}
					case "minecraft:basalt_deltas":
						if hash%37 == 0 {
							setGeneratedBlock(c, localX, y-1, localZ, magmaBlock)
						}
						if hash%173 == 0 {
							g.placeBasaltColumn(c, localX, y, localZ, 2+int((hash>>8)%7))
						}
					default:
						if hash%83 == 0 {
							setGeneratedBlock(c, localX, y, localZ, fireBlock)
						} else if hash%137 == 0 {
							setGeneratedBlock(c, localX, y, localZ, brownMushroomBlock)
						} else if hash%149 == 0 {
							setGeneratedBlock(c, localX, y, localZ, redMushroomBlock)
						}
					}
				}

				above := generatedBlock(c, localX, y+1, localZ).ResourceLocation()
				if above == "minecraft:netherrack" || above == "minecraft:basalt" || above == "minecraft:blackstone" {
					if hash%379 == 0 {
						g.placeGlowstoneBlob(c, localX, y, localZ, hash)
					} else if hash%211 == 0 {
						vine := weepingVinesBlock
						if g.BiomeAt3D(worldX, y, worldZ) == "minecraft:warped_forest" {
							vine = twistingVinesBlock
						}
						length := 1 + int((hash>>12)%5)
						for offset := 0; offset < length && generatedBlock(c, localX, y-offset, localZ).IsAir(); offset++ {
							setGeneratedBlock(c, localX, y-offset, localZ, vine)
						}
					}
				}
			}
		}
	}
}

func (g *NetherGenerator) placeHugeNetherFungus(c *Chunk, x, y, z int, warped bool, state uint64) {
	height := 6 + int((state>>16)%4)
	if y+height+2 >= netherMaxY {
		return
	}
	for offset := 0; offset < height; offset++ {
		if !generatedBlock(c, x, y+offset, z).IsAir() {
			return
		}
	}
	stem, wart := crimsonStemBlock, netherWartBlock
	if warped {
		stem, wart = warpedStemBlock, warpedWartBlock
	}
	for offset := 0; offset < height; offset++ {
		setGeneratedBlock(c, x, y+offset, z, stem)
	}
	capY := y + height - 2
	for dy := 0; dy <= 3; dy++ {
		radius := 2
		if dy == 0 || dy == 3 {
			radius = 1
		}
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				material := wart
				if (dx == 0 || dz == 0) && generatedHash(int64(state), dx, dy, dz)%9 == 0 {
					material = shroomlightBlock
				}
				setGeneratedBlock(c, x+dx, capY+dy, z+dz, material)
			}
		}
	}
}

func (g *NetherGenerator) placeBasaltColumn(c *Chunk, x, y, z, height int) {
	for offset := 0; offset < height && y+offset < netherMaxY; offset++ {
		if !generatedBlock(c, x, y+offset, z).IsAir() {
			break
		}
		setGeneratedBlock(c, x, y+offset, z, basaltBlock)
	}
}

func (g *NetherGenerator) placeGlowstoneBlob(c *Chunk, x, y, z int, state uint64) {
	setGeneratedBlock(c, x, y, z, glowstoneBlock)
	for i := 0; i < 18; i++ {
		dx := int(nextRandom(&state)*5) - 2
		dz := int(nextRandom(&state)*5) - 2
		dy := -int(nextRandom(&state) * 4)
		px, py, pz := x+dx, y+dy, z+dz
		if px < 0 || px >= SectionSize || pz < 0 || pz >= SectionSize || !generatedBlock(c, px, py, pz).IsAir() {
			continue
		}
		neighbours := 0
		for _, offset := range [][3]int{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
			nx, ny, nz := px+offset[0], py+offset[1], pz+offset[2]
			if nx >= 0 && nx < SectionSize && nz >= 0 && nz < SectionSize && generatedBlock(c, nx, ny, nz).ResourceLocation() == "minecraft:glowstone" {
				neighbours++
			}
		}
		if neighbours == 1 {
			setGeneratedBlock(c, px, py, pz, glowstoneBlock)
		}
	}
}

func (g *NetherGenerator) populateNetherBiomes(c *Chunk) {
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
