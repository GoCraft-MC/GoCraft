package world

import "testing"

func TestOverworldGeneratorDeterministicForSeed(t *testing.T) {
	first := NewOverworldGenerator(8675309).Generate(-3, 7)
	second := NewOverworldGenerator(8675309).Generate(-3, 7)
	for x := 0; x < SectionSize; x++ {
		for z := 0; z < SectionSize; z++ {
			if got, want := first.HighestBlockY(x, z), second.HighestBlockY(x, z); got != want {
				t.Fatalf("column (%d,%d) height = %d, want deterministic %d", x, z, got, want)
			}
			for y := WorldMinY; y <= first.HighestBlockY(x, z); y++ {
				if got, want := chunkBlock(first, x, y, z), chunkBlock(second, x, y, z); !got.Equal(want) {
					t.Fatalf("block (%d,%d,%d) = %s, want deterministic %s", x, y, z, got.Key(), want.Key())
				}
			}
		}
	}
}

func TestOverworldGeneratorSeedChangesTerrain(t *testing.T) {
	first := NewOverworldGenerator(1)
	second := NewOverworldGenerator(2)
	different := false
	for x := -256; x <= 256 && !different; x += 16 {
		for z := -256; z <= 256; z += 16 {
			if first.SurfaceHeight(x, z) != second.SurfaceHeight(x, z) {
				different = true
				break
			}
		}
	}
	if !different {
		t.Fatal("different seeds produced identical sampled terrain")
	}
}

func TestOverworldGeneratorProducesVariedLandAndOcean(t *testing.T) {
	generator := NewOverworldGenerator(123456789)
	heights := make(map[int]struct{})
	foundOcean := false
	for x := -640; x <= 640; x += 16 {
		for z := -640; z <= 640; z += 16 {
			height := generator.SurfaceHeight(x, z)
			heights[height] = struct{}{}
			if height < SeaLevel-2 {
				foundOcean = true
			}
			if height < 28 || height > 245 {
				t.Fatalf("surface height %d outside generator bounds", height)
			}
		}
	}
	if len(heights) < 16 {
		t.Fatalf("terrain has only %d sampled heights, want natural variation", len(heights))
	}
	if !foundOcean {
		t.Fatal("sampled terrain contains no ocean basin")
	}
}

func TestOverworldGeneratorLayersAndSafeSpawn(t *testing.T) {
	generator := NewOverworldGenerator(-42)
	chunk := generator.Generate(0, 0)
	solidSurface := generator.SurfaceHeight(0, 0)
	if solidSurface < SeaLevel+2 {
		t.Fatalf("origin surface = %d, want safe land above sea level", solidSurface)
	}
	if got := chunkBlock(chunk, 0, solidSurface, 0).ResourceLocation(); got != "minecraft:grass_block" {
		t.Fatalf("origin surface block = %s, want grass_block", got)
	}
	if got := chunkBlock(chunk, 0, solidSurface-1, 0).ResourceLocation(); got != "minecraft:dirt" {
		t.Fatalf("origin subsurface block = %s, want dirt", got)
	}
	if got := chunkBlock(chunk, 0, WorldMinY, 0).ResourceLocation(); got != "minecraft:bedrock" {
		t.Fatalf("bottom block = %s, want bedrock", got)
	}
	// HighestBlockY may be above solidSurface when vegetation (grass, tall plants)
	// is placed on top, so allow up to solidSurface+2 (one double-tall plant).
	if got := chunk.HighestBlockY(0, 0); got < solidSurface || got > solidSurface+2 {
		t.Fatalf("highest block = %d, want between solid surface %d and %d", got, solidSurface, solidSurface+2)
	}
}

func TestOverworldGeneratorFillsOceanToSeaLevel(t *testing.T) {
	generator := NewOverworldGenerator(20260731)
	for x := -1024; x <= 1024; x += 16 {
		for z := -1024; z <= 1024; z += 16 {
			solidSurface := generator.SurfaceHeight(x, z)
			if solidSurface >= SeaLevel-2 {
				continue
			}
			chunkX, localX := floorDiv16(x)
			chunkZ, localZ := floorDiv16(z)
			chunk := generator.Generate(int32(chunkX), int32(chunkZ))
			if got := chunkBlock(chunk, localX, SeaLevel, localZ).ResourceLocation(); got != "minecraft:water" {
				t.Fatalf("ocean surface block = %s, want water", got)
			}
			if got := chunk.HighestBlockY(localX, localZ); got != SeaLevel {
				t.Fatalf("ocean heightmap surface = %d, want sea level %d", got, SeaLevel)
			}
			return
		}
	}
	t.Fatal("could not find an ocean column in search area")
}

func TestOverworldGeneratorAddsDeterministicTrees(t *testing.T) {
	generator := NewOverworldGenerator(424242)
	foundLog := false
	foundLeaves := false
	for chunkX := int32(-4); chunkX <= 4 && !(foundLog && foundLeaves); chunkX++ {
		for chunkZ := int32(-4); chunkZ <= 4 && !(foundLog && foundLeaves); chunkZ++ {
			chunk := generator.Generate(chunkX, chunkZ)
			for _, section := range chunk.Sections {
				if section == nil {
					continue
				}
				for _, block := range section.BlockPalette() {
					switch block.ResourceLocation() {
					case "minecraft:oak_log":
						foundLog = true
					case "minecraft:oak_leaves":
						foundLeaves = true
					}
				}
			}
		}
	}
	if !foundLog || !foundLeaves {
		t.Fatalf("generated trees log=%v leaves=%v, want both", foundLog, foundLeaves)
	}
}
func chunkBlock(c *Chunk, x, y, z int) Block {
	sectionIndex := (y - WorldMinY) / SectionSize
	localY := (y - WorldMinY) % SectionSize
	if sectionIndex < 0 || sectionIndex >= SectionCount || c.Sections[sectionIndex] == nil {
		return Air
	}
	return c.Sections[sectionIndex].At(x, localY, z)
}

func floorDiv16(value int) (quotient, remainder int) {
	quotient = value / SectionSize
	remainder = value % SectionSize
	if remainder < 0 {
		quotient--
		remainder += SectionSize
	}
	return quotient, remainder
}
