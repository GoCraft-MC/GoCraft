package world

import (
	"sort"
	"testing"
)

func TestBiomeClassifierCoversJava1214OverworldPalette(t *testing.T) {
	type climate struct {
		height                        int
		temperature, humidity         float64
		continental, erosion          float64
		weirdness, peaks, river, rare float64
		mushroom                      bool
	}
	tests := map[string]climate{
		"deep_frozen_ocean":        {40, -0.7, 0, -0.5, 0, 0, 0, 0, 0, false},
		"deep_cold_ocean":          {40, -0.3, 0, -0.5, 0, 0, 0, 0, 0, false},
		"deep_ocean":               {40, 0, 0, -0.5, 0, 0, 0, 0, 0, false},
		"deep_lukewarm_ocean":      {40, 0.5, 0, -0.5, 0, 0, 0, 0, 0, false},
		"frozen_ocean":             {56, -0.7, 0, -0.3, 0, 0, 0, 0, 0, false},
		"cold_ocean":               {56, -0.3, 0, -0.3, 0, 0, 0, 0, 0, false},
		"ocean":                    {56, 0, 0, -0.3, 0, 0, 0, 0, 0, false},
		"lukewarm_ocean":           {56, 0.4, 0, -0.3, 0, 0, 0, 0, 0, false},
		"warm_ocean":               {56, 0.8, 0, -0.3, 0, 0, 0, 0, 0, false},
		"mushroom_fields":          {67, 0.2, 0.6, -0.1, 0, 0, 0, 0, 0.8, true},
		"river":                    {59, 0.1, 0, 0.2, 0, 0, 0, 1, 0, false},
		"frozen_river":             {59, -0.5, 0, 0.2, 0, 0, 0, 1, 0, false},
		"snowy_beach":              {64, -0.6, 0, 0, 0, 0, 0, 0, 0, false},
		"stony_shore":              {64, 0.1, 0, 0, 0, 0, 0.6, 0, 0, false},
		"beach":                    {64, 0.1, 0, 0, 0, 0, 0, 0, 0, false},
		"frozen_peaks":             {185, -0.3, 0, 0.6, 0, 0, 0.8, 0, 0, false},
		"stony_peaks":              {185, 0.2, 0, 0.6, 0, 0, 0.8, 0, 0, false},
		"jagged_peaks":             {155, -0.3, 0, 0.6, 0, 0, 0.7, 0, 0, false},
		"grove":                    {125, -0.5, 0.1, 0.5, 0, 0, 0.4, 0, 0, false},
		"snowy_slopes":             {125, -0.25, 0, 0.5, 0, 0, 0.4, 0, 0, false},
		"meadow":                   {125, 0.1, 0.2, 0.5, 0, 0, 0.4, 0, 0, false},
		"windswept_gravelly_hills": {125, 0.1, 0, 0.5, 0.5, 0, 0.4, 0, 0, false},
		"windswept_hills":          {125, 0.1, 0, 0.5, 0, 0, 0.4, 0, 0, false},
		"cherry_grove":             {85, 0.1, 0.3, 0.3, 0, 0, 0.2, 0, 0, false},
		"badlands":                 {80, 0.7, -0.7, 0.6, -0.3, 0, 0, 0, 0, false},
		"eroded_badlands":          {80, 0.7, -0.4, 0.6, -0.3, 0.6, 0, 0, 0, false},
		"wooded_badlands":          {80, 0.7, -0.4, 0.6, -0.3, 0, 0, 0, 0, false},
		"desert":                   {80, 0.7, -0.4, 0.2, 0, 0, 0, 0, 0, false},
		"savanna":                  {80, 0.4, 0, 0.2, 0, 0, 0, 0, 0, false},
		"savanna_plateau":          {80, 0.4, 0, 0.4, 0, 0, 0.5, 0, 0, false},
		"windswept_savanna":        {80, 0.4, 0, 0.4, 0, 0.6, 0.5, 0, 0, false},
		"swamp":                    {72, 0.1, 0.8, 0.1, 0, 0, 0, 0, 0, false},
		"mangrove_swamp":           {72, 0.4, 0.8, 0.1, 0, 0, 0, 0, 0, false},
		"bamboo_jungle":            {80, 0.5, 0.8, 0.3, 0, 0, 0, 0, 0, false},
		"jungle":                   {80, 0.5, 0.5, 0.3, 0, 0, 0, 0, 0, false},
		"sparse_jungle":            {80, 0.5, 0.5, 0.3, 0.6, 0, 0, 0, 0, false},
		"ice_spikes":               {80, -0.7, 0, 0.3, 0, 0, 0, 0, 0.7, false},
		"snowy_plains":             {80, -0.7, 0, 0.3, 0, 0, 0, 0, 0, false},
		"snowy_taiga":              {80, -0.3, -0.1, 0.3, 0, 0, 0, 0, 0, false},
		"old_growth_pine_taiga":    {80, -0.3, 0.4, 0.3, 0, -0.6, 0, 0, 0, false},
		"old_growth_spruce_taiga":  {80, -0.3, 0.4, 0.3, 0, 0.6, 0, 0, 0, false},
		"taiga":                    {80, -0.3, 0.1, 0.3, 0, 0, 0, 0, 0, false},
		"pale_garden":              {80, 0, 0.5, 0.3, 0.2, 0.2, 0, 0, 0.6, false},
		"dark_forest":              {80, 0, 0.5, 0.3, 0.2, 0, 0, 0, 0, false},
		"flower_forest":            {80, 0.2, 0.8, 0.3, 0, 0, 0, 0, 0, false},
		"forest":                   {80, 0.2, 0.5, 0.3, 0.2, 0, 0, 0, 0, false},
		"old_growth_birch_forest":  {80, 0.2, 0.4, 0.3, 0.2, 0, 0, 0, 0, false},
		"birch_forest":             {80, 0.2, 0.2, 0.3, 0, 0, 0, 0, 0, false},
		"windswept_forest":         {80, 0.1, 0.1, 0.4, 0.3, 0, 0.5, 0, 0, false},
		"sunflower_plains":         {80, 0.3, 0, 0.3, 0, 0, 0, 0, 0.7, false},
		"plains":                   {80, 0.3, 0, 0.3, 0, 0, 0, 0, 0, false},
	}

	seen := make(map[string]bool, len(tests))
	for want, c := range tests {
		got := chooseSurfaceBiome(c.height, c.temperature, c.humidity, c.continental, c.erosion, c.weirdness, c.peaks, c.river, c.rare, c.mushroom)
		if got != "minecraft:"+want {
			t.Errorf("classifier for %s returned %s", want, got)
		}
		seen[want] = true
	}
	for _, biome := range GeneratedBiomeNames() {
		if biome == "lush_caves" || biome == "dripstone_caves" || biome == "deep_dark" {
			continue
		}
		if !seen[biome] {
			t.Errorf("surface biome %s has no classifier coverage", biome)
		}
	}
}

func TestOverworldGeneratorHasThreeDimensionalCaveBiomes(t *testing.T) {
	generator := NewOverworldGenerator(20260808)
	caveBiomes := map[string]bool{}
	for chunkX := int32(-3); chunkX <= 3; chunkX++ {
		for chunkZ := int32(-3); chunkZ <= 3; chunkZ++ {
			chunk := generator.Generate(chunkX, chunkZ)
			for _, section := range chunk.Sections {
				if section == nil {
					continue
				}
				for _, biome := range section.BiomePalette() {
					switch biome {
					case "minecraft:lush_caves", "minecraft:dripstone_caves", "minecraft:deep_dark":
						caveBiomes[biome] = true
					}
				}
			}
		}
	}
	for _, biome := range []string{"minecraft:lush_caves", "minecraft:dripstone_caves", "minecraft:deep_dark"} {
		if !caveBiomes[biome] {
			t.Errorf("generated quart palettes contain no %s", biome)
		}
	}
}

func TestAquiferSamplerProducesWaterAndDeepLava(t *testing.T) {
	generator := NewOverworldGenerator(424242)
	if got := generator.aquiferMaterial(0, -55, 0).ResourceLocation(); got != "minecraft:lava" {
		t.Fatalf("deep aquifer material = %s, want lava", got)
	}
	foundWater := false
	for x := -512; x <= 512 && !foundWater; x += 8 {
		for z := -512; z <= 512; z += 8 {
			for y := -36; y <= 20; y += 4 {
				if generator.aquiferMaterial(x, y, z).ResourceLocation() == "minecraft:water" {
					foundWater = true
					break
				}
			}
		}
	}
	if !foundWater {
		t.Fatal("aquifer sampler produced no underground water")
	}
}

func TestOverworldGeneratorSamplesModernSurfaceBiomeFamilies(t *testing.T) {
	generator := NewOverworldGenerator(123456789)
	seen := map[string]bool{}
	for x := -8192; x <= 8192; x += 64 {
		for z := -8192; z <= 8192; z += 64 {
			seen[generator.BiomeAt(x, z)] = true
		}
	}
	for _, biome := range GeneratedBiomeNames() {
		if biome == "lush_caves" || biome == "dripstone_caves" || biome == "deep_dark" {
			continue
		}
		name := "minecraft:" + biome
		if !seen[name] {
			t.Errorf("sample contains no %s", name)
		}
	}
	biomes := make([]string, 0, len(seen))
	for biome := range seen {
		biomes = append(biomes, biome)
	}
	sort.Strings(biomes)
	t.Logf("sampled %d surface biomes: %v", len(biomes), biomes)
}

func TestGeneratedCavesContainAquifersAndBiomeMaterials(t *testing.T) {
	generator := NewOverworldGenerator(20260808)
	found := map[string]bool{}
	for chunkX := int32(-4); chunkX <= 4; chunkX++ {
		for chunkZ := int32(-4); chunkZ <= 4; chunkZ++ {
			chunk := generator.Generate(chunkX, chunkZ)
			for localX := 0; localX < SectionSize; localX++ {
				worldX := int(chunkX)*SectionSize + localX
				for localZ := 0; localZ < SectionSize; localZ++ {
					worldZ := int(chunkZ)*SectionSize + localZ
					surfaceY := generator.SurfaceHeight(worldX, worldZ)
					for y := WorldMinY + 5; y < minInt(surfaceY-6, 72); y++ {
						switch chunkBlock(chunk, localX, y, localZ).ResourceLocation() {
						case "minecraft:lava":
							if y <= -55 {
								found["lava"] = true
							}
						case "minecraft:water":
							if y < 45 && surfaceY > SeaLevel+4 {
								found["water"] = true
							}
						case "minecraft:moss_block":
							found["lush"] = true
						case "minecraft:dripstone_block", "minecraft:pointed_dripstone":
							found["dripstone"] = true
						case "minecraft:sculk":
							found["deep_dark"] = true
						}
					}
				}
			}
		}
	}
	for _, feature := range []string{"lava", "water", "lush", "dripstone", "deep_dark"} {
		if !found[feature] {
			t.Errorf("sampled caves contain no %s feature", feature)
		}
	}
}
