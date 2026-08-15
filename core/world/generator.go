package world

import "math"

// SeaLevel is the Java overworld sea level.
const SeaLevel = 63

// Generator creates chunks for positions not yet loaded or generated.
// Implementations must be safe for concurrent calls.
type Generator interface {
	Generate(x, z int32) *Chunk
}

// OverworldGenerator creates deterministic, seed-backed overworld terrain.
// Generation follows the modern vanilla/Pumpkin staging model: continental
// terrain and climate, 3-D biome selection, surface rules, noise and worm
// carvers, aquifers, ores, structures, and biome decoration.
type OverworldGenerator struct {
	seed int64
}

func NewOverworldGenerator(seed int64) *OverworldGenerator { return &OverworldGenerator{seed: seed} }
func (g *OverworldGenerator) Seed() int64                  { return g.seed }

var generatedBiomeNames = []string{
	"badlands",
	"bamboo_jungle",
	"beach",
	"birch_forest",
	"cherry_grove",
	"cold_ocean",
	"dark_forest",
	"deep_cold_ocean",
	"deep_dark",
	"deep_frozen_ocean",
	"deep_lukewarm_ocean",
	"deep_ocean",
	"desert",
	"dripstone_caves",
	"eroded_badlands",
	"flower_forest",
	"forest",
	"frozen_ocean",
	"frozen_peaks",
	"frozen_river",
	"grove",
	"ice_spikes",
	"jagged_peaks",
	"jungle",
	"lukewarm_ocean",
	"lush_caves",
	"mangrove_swamp",
	"meadow",
	"mushroom_fields",
	"ocean",
	"old_growth_birch_forest",
	"old_growth_pine_taiga",
	"old_growth_spruce_taiga",
	"pale_garden",
	"plains",
	"river",
	"savanna",
	"savanna_plateau",
	"snowy_beach",
	"snowy_plains",
	"snowy_slopes",
	"snowy_taiga",
	"sparse_jungle",
	"stony_peaks",
	"stony_shore",
	"sunflower_plains",
	"swamp",
	"taiga",
	"warm_ocean",
	"windswept_forest",
	"windswept_gravelly_hills",
	"windswept_hills",
	"windswept_savanna",
	"wooded_badlands",
}

// GeneratedBiomeNames returns every biome name that the GoCraft overworld
// generator can produce, without the minecraft namespace prefix.
func GeneratedBiomeNames() []string {
	return append([]string(nil), generatedBiomeNames...)
}

func block(name string) Block { return Block{Namespace: "minecraft", Name: name} }

func blockProps(name string, kvPairs ...string) Block {
	b := Block{Namespace: "minecraft", Name: name}
	if len(kvPairs) >= 2 {
		b.Properties = make(map[string]string)
		for i := 0; i+1 < len(kvPairs); i += 2 {
			b.Properties[kvPairs[i]] = kvPairs[i+1]
		}
	}
	return b
}

var (
	stoneBlock                = block("stone")
	deepslateBlock            = block("deepslate")
	grassBlock                = block("grass_block")
	dirtBlock                 = block("dirt")
	bedrockBlock              = block("bedrock")
	sandBlock                 = block("sand")
	sandstoneBlock            = block("sandstone")
	redSandBlock              = block("red_sand")
	terracottaBlock           = block("terracotta")
	orangeTerracottaBlock     = block("orange_terracotta")
	yellowTerracottaBlock     = block("yellow_terracotta")
	brownTerracottaBlock      = block("brown_terracotta")
	whiteTerracottaBlock      = block("white_terracotta")
	lightGrayTerracottaBlock  = block("light_gray_terracotta")
	gravelBlock               = block("gravel")
	coarseDirtBlock           = block("coarse_dirt")
	mudBlock                  = block("mud")
	podzolBlock               = block("podzol")
	waterBlock                = block("water")
	lavaBlock                 = block("lava")
	iceBlock                  = block("ice")
	snowBlock                 = block("snow_block")
	myceliumBlock             = block("mycelium")
	packedIceBlock            = block("packed_ice")
	blueIceBlock              = block("blue_ice")
	mossBlock                 = block("moss_block")
	clayBlock                 = block("clay")
	dripstoneBlock            = block("dripstone_block")
	pointedDripstoneUpBlock   = blockProps("pointed_dripstone", "vertical_direction", "up", "thickness", "tip", "waterlogged", "false")
	pointedDripstoneDownBlock = blockProps("pointed_dripstone", "vertical_direction", "down", "thickness", "tip", "waterlogged", "false")
	sculkBlock                = block("sculk")
	oakLogBlock               = block("oak_log")
	oakLeafBlock              = block("oak_leaves")
	birchLogBlock             = block("birch_log")
	birchLeafBlock            = block("birch_leaves")
	spruceLogBlock            = block("spruce_log")
	spruceLeafBlock           = block("spruce_leaves")
	acaciaLogBlock            = block("acacia_log")
	acaciaLeafBlock           = block("acacia_leaves")
	jungleLogBlock            = block("jungle_log")
	jungleLeafBlock           = block("jungle_leaves")
	mangroveLogBlock          = block("mangrove_log")
	mangroveLeafBlock         = block("mangrove_leaves")
	cactusBlock               = block("cactus")
	coalOreBlock              = block("coal_ore")
	deepslateCoalOreBlock     = block("deepslate_coal_ore")
	ironOreBlock              = block("iron_ore")
	deepslateIronOreBlock     = block("deepslate_iron_ore")
	copperOreBlock            = block("copper_ore")
	deepslateCopperOreBlock   = block("deepslate_copper_ore")
	goldOreBlock              = block("gold_ore")
	deepslateGoldOreBlock     = block("deepslate_gold_ore")
	redstoneOreBlock          = block("redstone_ore")
	deepslateRedstoneOreBlock = block("deepslate_redstone_ore")
	lapisOreBlock             = block("lapis_ore")
	deepslateLapisOreBlock    = block("deepslate_lapis_ore")
	diamondOreBlock           = block("diamond_ore")
	deepslateDiamondOreBlock  = block("deepslate_diamond_ore")
	emeraldOreBlock           = block("emerald_ore")
	deepslateEmeraldOreBlock  = block("deepslate_emerald_ore")

	// Ground cover — grass types
	shortGrassBlock     = block("short_grass")
	fernBlock           = block("fern")
	tallGrassLowerBlock = blockProps("tall_grass", "half", "lower")
	tallGrassUpperBlock = blockProps("tall_grass", "half", "upper")
	largeFernLowerBlock = blockProps("large_fern", "half", "lower")
	largeFernUpperBlock = blockProps("large_fern", "half", "upper")

	// Single-block flowers
	dandelionBlock       = block("dandelion")
	poppyBlock           = block("poppy")
	alliumBlock          = block("allium")
	azureBluetBlock      = block("azure_bluet")
	redTulipBlock        = block("red_tulip")
	orangeTulipBlock     = block("orange_tulip")
	whiteTulipBlock      = block("white_tulip")
	pinkTulipBlock       = block("pink_tulip")
	oxeyeDaisyBlock      = block("oxeye_daisy")
	cornflowerBlock      = block("cornflower")
	lilyOfTheValleyBlock = block("lily_of_the_valley")
	blueOrchidBlock      = block("blue_orchid")

	// Double-block tall flowers
	sunflowerLowerBlock = blockProps("sunflower", "half", "lower")
	sunflowerUpperBlock = blockProps("sunflower", "half", "upper")
	lilacLowerBlock     = blockProps("lilac", "half", "lower")
	lilacUpperBlock     = blockProps("lilac", "half", "upper")
	roseBushLowerBlock  = blockProps("rose_bush", "half", "lower")
	roseBushUpperBlock  = blockProps("rose_bush", "half", "upper")
	peonyLowerBlock     = blockProps("peony", "half", "lower")
	peonyUpperBlock     = blockProps("peony", "half", "upper")

	// Other vegetation
	deadBushBlock      = block("dead_bush")
	sugarCaneBlock     = block("sugar_cane")
	bambooBlock        = block("bamboo")
	lilyPadBlock       = block("lily_pad")
	seagrassBlock      = block("seagrass")
	brownMushroomBlock = block("brown_mushroom")
	redMushroomBlock   = block("red_mushroom")
	tubeCoralBlock     = block("tube_coral_block")
	brainCoralBlock    = block("brain_coral_block")
	bubbleCoralBlock   = block("bubble_coral_block")
	fireCoralBlock     = block("fire_coral_block")
	hornCoralBlock     = block("horn_coral_block")

	// Additional tree types
	darkOakLogBlock  = block("dark_oak_log")
	darkOakLeafBlock = block("dark_oak_leaves")
	cherryLogBlock   = block("cherry_log")
	cherryLeafBlock  = block("cherry_leaves")
	paleOakLogBlock  = block("pale_oak_log")
	paleOakLeafBlock = block("pale_oak_leaves")
)

type terrainSample struct {
	height        int
	biome         string
	temperature   float64
	humidity      float64
	continental   float64
	erosion       float64
	weirdness     float64
	peakStrength  float64
	riverStrength float64
}

// Generate creates a full 1.18+ height chunk and applies features in a stable
// order so the same seed and coordinate always produce identical bytes.
func (g *OverworldGenerator) Generate(chunkX, chunkZ int32) *Chunk {
	c := &Chunk{X: chunkX, Z: chunkZ}
	var heights [SectionSize * SectionSize]int

	for localX := 0; localX < SectionSize; localX++ {
		for localZ := 0; localZ < SectionSize; localZ++ {
			worldX := int(chunkX)*SectionSize + localX
			worldZ := int(chunkZ)*SectionSize + localZ
			sample := g.sampleTerrain(worldX, worldZ)
			surfaceY := sample.height
			heights[localZ*SectionSize+localX] = surfaceY
			underwater := surfaceY < SeaLevel

			for y := WorldMinY; y <= surfaceY; y++ {
				material := stoneBlock
				if y < 0 {
					material = deepslateBlock
				}
				switch {
				case y == WorldMinY:
					material = bedrockBlock
				case y < WorldMinY+5 && g.bedrockAt(worldX, y, worldZ):
					material = bedrockBlock
				case surfaceY-y <= 5:
					material = g.surfaceMaterial(sample.biome, y, surfaceY-y, underwater, worldX, worldZ)
				}
				setGeneratedBlock(c, localX, y, localZ, material)
			}

			for y := surfaceY + 1; y <= SeaLevel; y++ {
				material := waterBlock
				if y == SeaLevel && isFrozenBiome(sample.biome) {
					material = iceBlock
				}
				setGeneratedBlock(c, localX, y, localZ, material)
			}
		}
	}

	g.carveCaves(c, heights)
	g.carveNoiseCaves(c, heights)
	g.addOres(c)
	g.decorateCaves(c, heights)
	g.addBiomeLandmarks(c)
	g.addVillageStructures(c)
	g.addVegetation(c)
	g.addGroundCover(c, heights)
	g.populateBiomes(c)
	return c
}

// SurfaceHeight returns the highest terrain block before vegetation.
func (g *OverworldGenerator) SurfaceHeight(x, z int) int { return g.sampleTerrain(x, z).height }

// BiomeAt returns the deterministic surface biome at an absolute column.
func (g *OverworldGenerator) BiomeAt(x, z int) string { return g.sampleTerrain(x, z).biome }

// BiomeAt3D returns the biome at an absolute block position. Modern Java
// worlds store biomes in 4x4x4 cells, allowing cave biomes below a different
// surface biome in the same column.
func (g *OverworldGenerator) BiomeAt3D(x, y, z int) string {
	return g.biomeAt3D(x, y, z, g.sampleTerrain(x, z))
}

// NearestBiome finds a nearby sample of target within maxDistance blocks.
// Samples are spaced 32 blocks apart, matching the broad scale of GoCraft's
// climate regions while keeping an in-game lookup inexpensive.
func (g *OverworldGenerator) NearestBiome(x, z int, target string, maxDistance int) (int, int, bool) {
	if maxDistance < 0 {
		return 0, 0, false
	}
	const sampleStep = 32
	for radius := 0; radius <= maxDistance; radius += sampleStep {
		bestX, bestZ := 0, 0
		bestDistanceSquared := int64(math.MaxInt64)
		found := false
		consider := func(sampleX, sampleZ int) {
			surface := g.sampleTerrain(sampleX, sampleZ)
			matches := surface.biome == target
			if isCaveBiome(target) {
				matches = false
				maxY := minInt(56, surface.height-8)
				for sampleY := WorldMinY + 8; sampleY <= maxY; sampleY += 12 {
					if g.biomeAt3D(sampleX, sampleY, sampleZ, surface) == target {
						matches = true
						break
					}
				}
			}
			if !matches {
				return
			}
			dx := int64(sampleX - x)
			dz := int64(sampleZ - z)
			distanceSquared := dx*dx + dz*dz
			if distanceSquared < bestDistanceSquared {
				bestX, bestZ = sampleX, sampleZ
				bestDistanceSquared = distanceSquared
				found = true
			}
		}

		if radius == 0 {
			consider(x, z)
		} else {
			for offset := -radius; offset <= radius; offset += sampleStep {
				consider(x+offset, z-radius)
				consider(x+offset, z+radius)
				if offset != -radius && offset != radius {
					consider(x-radius, z+offset)
					consider(x+radius, z+offset)
				}
			}
		}
		if found {
			return bestX, bestZ, true
		}
	}
	return 0, 0, false
}

func isCaveBiome(biome string) bool {
	switch biome {
	case "minecraft:lush_caves", "minecraft:dripstone_caves", "minecraft:deep_dark":
		return true
	default:
		return false
	}
}

func (g *OverworldGenerator) sampleTerrain(x, z int) terrainSample {
	fx, fz := float64(x), float64(z)

	// Domain-warp the continental coordinates so continent shapes are irregular.
	warpX := g.fractal(fx, fz, 700, 3, 0.5, 0x7761727058585858) * 220
	warpZ := g.fractal(fx, fz, 700, 3, 0.5, 0x776172705a5a5a5a) * 220
	continental := g.fractal(fx+warpX, fz+warpZ, 1800, 5, 0.54, 0x636f6e74696e656e)

	erosion := g.fractal(fx, fz, 600, 4, 0.52, 0x65726f73696f6e31)
	ridgeNoise := g.fractal(fx, fz, 500, 4, 0.53, 0x7269646765733031)
	weirdness := g.fractal(fx, fz, 260, 3, 0.5, 0x77656972646e6573)
	detail := g.fractal(fx, fz, 80, 4, 0.46, 0x64657461696c3031)
	temperature := g.fractal(fx, fz, 700, 4, 0.52, 0x74656d7065726174)
	humidity := g.fractal(fx, fz, 650, 4, 0.52, 0x68756d6964697479)
	riverNoise := g.fractal(fx+warpX*0.3, fz+warpZ*0.3, 310, 3, 0.5, 0x7269766572733031)
	rareBiomeNoise := g.fractal(fx, fz, 1050, 3, 0.5, 0x7261726562696f6d)

	// Bimodal height: land clearly above sea level, ocean clearly below.
	// continental ≥ landThreshold → land (~60% of terrain), else → ocean.
	ridge := 1 - math.Abs(ridgeNoise)
	land := clamp01((continental + 0.15) / 0.85)
	peakStrength := clamp01((ridge-0.25)/0.75) * math.Pow(land, 1.1)

	const landThreshold = -0.1
	var base float64
	if continental >= landThreshold {
		// Land: base ranges 64–100, shaped by how far above threshold we are
		// and pushed down by high erosion (flat plains) or up by low erosion (hills).
		t := (continental - landThreshold) / (1.0 - landThreshold)
		base = 64 + t*36 + erosion*(-14) + detail*6
	} else {
		// Ocean: base ranges 30–52, deeper for more negative continental.
		t := (landThreshold - continental) / (landThreshold + 1.0)
		base = 52 - t*22 + erosion*3 + detail*2
	}
	peaks := math.Pow(peakStrength, 1.1) * (110 + math.Max(0, weirdness)*80)
	height := int(math.Round(base + peaks))

	// Modern rivers follow narrow zero contours through continental terrain.
	// Lowering the density gradually around the contour produces a valley and
	// natural banks instead of painting a river biome over unchanged terrain.
	riverStrength := clamp01((0.052-math.Abs(riverNoise))/0.052) * clamp01((continental+0.12)/0.32)
	if riverStrength > 0 && height < 138 {
		riverBed := float64(SeaLevel - 4)
		height = int(math.Round(lerp(float64(height), riverBed, math.Pow(riverStrength, 0.72))))
	}

	// Mushroom fields form rare islands just outside ordinary coastlines.
	mushroomIsland := rareBiomeNoise > 0.68 && continental > -0.22 && continental < 0.02
	if mushroomIsland && height < SeaLevel+4 {
		height = SeaLevel + 4 + int((rareBiomeNoise-0.68)*25)
		riverStrength = 0
	}

	// Keep a large landmass around spawn so players never start in the ocean.
	distance := math.Hypot(fx, fz)
	if distance < 200 {
		minimum := float64(SeaLevel+6) - distance/36
		if float64(height) < minimum {
			height = int(math.Round(minimum))
		}
		riverStrength = 0
	}
	if height < 28 {
		height = 28
	}
	if height > 245 {
		height = 245
	}

	biome := chooseSurfaceBiome(height, temperature, humidity, continental, erosion, weirdness, peakStrength, riverStrength, rareBiomeNoise, mushroomIsland)
	return terrainSample{
		height: height, biome: biome, temperature: temperature, humidity: humidity,
		continental: continental, erosion: erosion, weirdness: weirdness,
		peakStrength: peakStrength, riverStrength: riverStrength,
	}
}

func chooseBiome(height int, temperature, humidity, continental, erosion, peaks float64) string {
	return chooseSurfaceBiome(height, temperature, humidity, continental, erosion, 0, peaks, 0, 0, false)
}

func chooseSurfaceBiome(height int, temperature, humidity, continental, erosion, weirdness, peaks, riverStrength, rareNoise float64, mushroomIsland bool) string {
	if mushroomIsland {
		return "minecraft:mushroom_fields"
	}
	if riverStrength > 0.52 && continental >= -0.1 && height <= SeaLevel+2 {
		if temperature < -0.32 {
			return "minecraft:frozen_river"
		}
		return "minecraft:river"
	}
	// Deep ocean
	if height < SeaLevel-13 {
		switch {
		case temperature < -0.45:
			return "minecraft:deep_frozen_ocean"
		case temperature < -0.12:
			return "minecraft:deep_cold_ocean"
		case temperature > 0.32:
			return "minecraft:deep_lukewarm_ocean"
		default:
			return "minecraft:deep_ocean"
		}
	}
	// Shallow oceans use the same five temperature bands as Java 1.21.4.
	if height < SeaLevel-2 {
		switch {
		case temperature < -0.45:
			return "minecraft:frozen_ocean"
		case temperature < -0.12:
			return "minecraft:cold_ocean"
		case temperature > 0.62:
			return "minecraft:warm_ocean"
		case temperature > 0.25:
			return "minecraft:lukewarm_ocean"
		default:
			return "minecraft:ocean"
		}
	}
	// Coastline
	if height <= SeaLevel+2 {
		if temperature < -0.4 {
			return "minecraft:snowy_beach"
		}
		if peaks > 0.18 || erosion < -0.12 {
			return "minecraft:stony_shore"
		}
		return "minecraft:beach"
	}
	// High mountain peaks
	if height >= 178 {
		if temperature < -0.18 {
			return "minecraft:frozen_peaks"
		}
		return "minecraft:stony_peaks"
	}
	if height >= 145 {
		if temperature < -0.18 {
			return "minecraft:jagged_peaks"
		}
		return "minecraft:stony_peaks"
	}
	// Upper slopes
	if height >= 116 {
		if temperature < -0.38 && humidity > -0.2 {
			return "minecraft:grove"
		}
		if temperature < -0.2 {
			return "minecraft:snowy_slopes"
		}
		if humidity > 0.05 {
			return "minecraft:meadow"
		}
		if erosion > 0.35 {
			return "minecraft:windswept_gravelly_hills"
		}
		return "minecraft:windswept_hills"
	}
	// Cherry grove: moderate slopes with mild temperature and moderate humidity
	if height >= 80 && temperature > -0.08 && temperature < 0.32 && humidity > 0.2 && humidity < 0.6 && erosion < 0.1 {
		return "minecraft:cherry_grove"
	}
	// Hot & dry
	if temperature > 0.48 && humidity < -0.22 {
		if continental > 0.2 {
			if weirdness > 0.36 && erosion < 0.2 {
				return "minecraft:eroded_badlands"
			}
			if humidity > -0.52 || erosion > 0.18 {
				return "minecraft:wooded_badlands"
			}
			return "minecraft:badlands"
		}
		return "minecraft:desert"
	}
	// Savanna
	if temperature > 0.32 && humidity < 0.18 {
		if peaks > 0.32 || erosion < -0.48 {
			if weirdness > 0.42 {
				return "minecraft:windswept_savanna"
			}
			return "minecraft:savanna_plateau"
		}
		return "minecraft:savanna"
	}
	// Swamp variants
	if humidity > 0.62 && height < 76 {
		if temperature > 0.28 {
			return "minecraft:mangrove_swamp"
		}
		return "minecraft:swamp"
	}
	// Jungle variants
	if temperature > 0.38 && humidity > 0.38 {
		if humidity > 0.65 {
			return "minecraft:bamboo_jungle"
		}
		if erosion > 0.42 {
			return "minecraft:sparse_jungle"
		}
		return "minecraft:jungle"
	}
	// Cold / snowy
	if temperature < -0.48 {
		if rareNoise > 0.48 && erosion < 0.15 {
			return "minecraft:ice_spikes"
		}
		return "minecraft:snowy_plains"
	}
	if temperature < -0.2 {
		if humidity > 0.28 {
			if weirdness > 0.38 {
				return "minecraft:old_growth_spruce_taiga"
			}
			if weirdness < -0.38 {
				return "minecraft:old_growth_pine_taiga"
			}
		}
		if humidity < -0.05 {
			return "minecraft:snowy_taiga"
		}
		return "minecraft:taiga"
	}
	// Temperate forests
	if humidity > 0.42 {
		if temperature < 0.08 {
			if rareNoise > 0.48 && weirdness > 0.08 {
				return "minecraft:pale_garden"
			}
			return "minecraft:dark_forest"
		}
		if humidity > 0.68 {
			return "minecraft:flower_forest"
		}
		return "minecraft:forest"
	}
	if humidity > 0.12 && temperature < 0.25 {
		if humidity > 0.34 {
			return "minecraft:old_growth_birch_forest"
		}
		return "minecraft:birch_forest"
	}
	if peaks > 0.28 && erosion > 0.18 {
		if humidity > 0.08 {
			return "minecraft:windswept_forest"
		}
		return "minecraft:windswept_hills"
	}
	if rareNoise > 0.5 && humidity > -0.15 {
		return "minecraft:sunflower_plains"
	}
	return "minecraft:plains"
}

func (g *OverworldGenerator) surfaceMaterial(biome string, y, depth int, underwater bool, x, z int) Block {
	if underwater {
		if depth <= 3 {
			if biome == "minecraft:river" || biome == "minecraft:frozen_river" || g.columnHash(x, z, 0x67726176656c)%5 == 0 {
				return gravelBlock
			}
			return sandBlock
		}
		return sandstoneBlock
	}
	switch biome {
	case "minecraft:desert", "minecraft:beach", "minecraft:snowy_beach":
		if depth <= 3 {
			return sandBlock
		}
		return sandstoneBlock
	case "minecraft:badlands", "minecraft:eroded_badlands":
		if depth == 0 {
			return redSandBlock
		}
		return g.badlandsBand(y, x, z)
	case "minecraft:wooded_badlands":
		if depth == 0 && y > 84 {
			if g.columnHash(x, z, 0x776f6f646261646c)%3 == 0 {
				return coarseDirtBlock
			}
			return grassBlock
		}
		if depth == 0 {
			return redSandBlock
		}
		return g.badlandsBand(y, x, z)
	case "minecraft:swamp", "minecraft:mangrove_swamp":
		if depth == 0 {
			return grassBlock
		}
		return mudBlock
	case "minecraft:mushroom_fields":
		if depth == 0 {
			return myceliumBlock
		}
		return dirtBlock
	case "minecraft:taiga", "minecraft:old_growth_pine_taiga", "minecraft:old_growth_spruce_taiga":
		if depth == 0 {
			if g.columnHash(x, z, 0x706f647a6f6c3031)%5 == 0 {
				return podzolBlock
			}
			return grassBlock
		}
		return dirtBlock
	case "minecraft:dark_forest", "minecraft:pale_garden":
		if depth == 0 {
			return grassBlock
		}
		return dirtBlock
	case "minecraft:windswept_gravelly_hills":
		if depth <= 2 && g.columnHash(x, z, 0x67726176656c6c79)%3 != 0 {
			return gravelBlock
		}
		return stoneBlock
	case "minecraft:stony_shore", "minecraft:windswept_hills", "minecraft:stony_peaks", "minecraft:jagged_peaks":
		return stoneBlock
	case "minecraft:snowy_plains", "minecraft:snowy_slopes", "minecraft:snowy_taiga", "minecraft:grove", "minecraft:ice_spikes", "minecraft:frozen_peaks":
		if depth == 0 {
			return snowBlock
		}
		if depth <= 3 {
			return dirtBlock
		}
		return stoneBlock
	default:
		if depth == 0 {
			return grassBlock
		}
		return dirtBlock
	}
}

func (g *OverworldGenerator) badlandsBand(y, x, z int) Block {
	shift := int(g.columnHash(floorDiv(x, 4), floorDiv(z, 4), 0x6261646c616e6473)%7) - 3
	switch floorMod(y+shift, 17) {
	case 0, 1:
		return orangeTerracottaBlock
	case 4:
		return yellowTerracottaBlock
	case 8:
		return brownTerracottaBlock
	case 12:
		return whiteTerracottaBlock
	case 13:
		return lightGrayTerracottaBlock
	default:
		return terracottaBlock
	}
}

func isFrozenBiome(biome string) bool {
	switch biome {
	case "minecraft:frozen_ocean", "minecraft:deep_frozen_ocean", "minecraft:frozen_river", "minecraft:snowy_beach":
		return true
	default:
		return false
	}
}

func setGeneratedBlock(c *Chunk, x, y, z int, material Block) {
	if y < WorldMinY || y > WorldMaxY {
		return
	}
	sectionIndex := (y - WorldMinY) / SectionSize
	localY := (y - WorldMinY) % SectionSize
	if c.Sections[sectionIndex] == nil {
		c.Sections[sectionIndex] = NewSection()
	}
	c.Sections[sectionIndex].Set(x, localY, z, material)
}

func generatedBlock(c *Chunk, x, y, z int) Block {
	if y < WorldMinY || y > WorldMaxY {
		return Air
	}
	sectionIndex := (y - WorldMinY) / SectionSize
	localY := (y - WorldMinY) % SectionSize
	if c.Sections[sectionIndex] == nil {
		return Air
	}
	return c.Sections[sectionIndex].At(x, localY, z)
}

func (g *OverworldGenerator) bedrockAt(x, y, z int) bool {
	layer := y - WorldMinY
	return int(g.columnHash(x, z, uint64(y))%5) >= layer
}

func (g *OverworldGenerator) fractal(x, z, scale float64, octaves int, persistence float64, salt uint64) float64 {
	amplitude, totalAmplitude, value := 1.0, 0.0, 0.0
	for octave := 0; octave < octaves; octave++ {
		value += g.gradNoise(x/scale, z/scale, salt+uint64(octave)*0x9e3779b97f4a7c15) * amplitude
		totalAmplitude += amplitude
		amplitude *= persistence
		scale *= 0.5
	}
	return value / totalAmplitude
}

// gradNoise is 2-D gradient (Perlin-style) noise. Unlike value noise, it uses
// random gradient vectors at each lattice corner so the output is smooth and
// organic rather than blobby/circular.
func (g *OverworldGenerator) gradNoise(x, z float64, salt uint64) float64 {
	x0, z0 := int64(math.Floor(x)), int64(math.Floor(z))
	dx, dz := x-float64(x0), z-float64(z0)
	tx, tz := smoothstep(dx), smoothstep(dz)

	g00 := g.grad2D(x0, z0, salt, dx, dz)
	g10 := g.grad2D(x0+1, z0, salt, dx-1, dz)
	g01 := g.grad2D(x0, z0+1, salt, dx, dz-1)
	g11 := g.grad2D(x0+1, z0+1, salt, dx-1, dz-1)

	// Gradient noise has a narrower output range than value noise (~0.7 max).
	// Scale up to restore the [-1,1] range expected by terrain callers.
	return lerp(lerp(g00, g10, tx), lerp(g01, g11, tx), tz) * 1.41
}

// grad2D returns the dot product of a pseudorandom unit gradient with (dx,dz).
// The 8 cardinal/diagonal directions cover the full unit circle uniformly.
func (g *OverworldGenerator) grad2D(x, z int64, salt uint64, dx, dz float64) float64 {
	h := mix64(uint64(g.seed) ^ uint64(x)*0x9e3779b185ebca87 ^ uint64(z)*0xc2b2ae3d27d4eb4f ^ salt)
	switch h & 7 {
	case 0:
		return dx + dz
	case 1:
		return -dx + dz
	case 2:
		return dx - dz
	case 3:
		return -dx - dz
	case 4:
		return dx*1.4 + dz*0.7
	case 5:
		return -dx*1.4 + dz*0.7
	case 6:
		return dx*0.7 + dz*1.4
	default:
		return dx*0.7 - dz*1.4
	}
}

func (g *OverworldGenerator) columnHash(x, z int, salt uint64) uint64 {
	return mix64(uint64(g.seed) ^ uint64(int64(x))*0x517cc1b727220a95 ^ uint64(int64(z))*0x6eed0e9da4d94a4f ^ salt)
}

func (g *OverworldGenerator) featureHash(x, z int32, salt uint64) uint64 {
	return mix64(uint64(g.seed) ^ uint64(int64(x))*0x8da6b343 ^ uint64(int64(z))*0xd8163841 ^ salt)
}

func mix64(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}

func nextRandom(state *uint64) float64 {
	*state = mix64(*state)
	return float64(*state>>11) / float64(uint64(1)<<53)
}

func floorDiv(value, divisor int) int {
	quotient := value / divisor
	if value%divisor < 0 {
		quotient--
	}
	return quotient
}

func floorMod(value, divisor int) int {
	remainder := value % divisor
	if remainder < 0 {
		remainder += divisor
	}
	return remainder
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func smoothstep(v float64) float64 { return v * v * (3 - 2*v) }
func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// FlatGenerator remains available for adapter tests.
type FlatGenerator struct{}

func (g *FlatGenerator) Generate(x, z int32) *Chunk {
	c := &Chunk{X: x, Z: z}
	const groundSectionIdx = 7
	const groundLocalY = 15
	sec := NewSection()
	sec.SetUniformBiome("minecraft:plains")
	for bx := 0; bx < SectionSize; bx++ {
		for bz := 0; bz < SectionSize; bz++ {
			sec.Set(bx, groundLocalY, bz, stoneBlock)
		}
	}
	c.Sections[groundSectionIdx] = sec
	return c
}

func setChunkBiome(c *Chunk, biome string) {
	for index, section := range c.Sections {
		if section == nil {
			continue
		}
		section.SetUniformBiome(biome)
		c.Sections[index] = section
	}
}

func generatedHash(seed int64, x, y, z int) uint64 {
	value := uint64(seed) ^ uint64(int64(x))*0x4f1bbcdc6762fda9 ^ uint64(int64(y))*0x632be59bd9b4e019 ^ uint64(int64(z))*0x9e3779b97f4a7c15
	value ^= value >> 30
	value *= 0xbf58476d1ce4e5b9
	value ^= value >> 27
	value *= 0x94d049bb133111eb
	return value ^ (value >> 31)
}
