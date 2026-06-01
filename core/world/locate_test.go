package world

import (
	"strings"
	"testing"
)

func TestNearestVillageMatchesGeneratedVillagePlacement(t *testing.T) {
	generator := NewOverworldGenerator(0)
	center, ok := generator.NearestVillage(0, 0, 8192)
	if !ok {
		t.Fatal("expected a village within 8192 blocks of spawn")
	}
	if !isVillageBiome(center.Biome) {
		t.Fatalf("located village biome = %q", center.Biome)
	}
	height := generator.SurfaceHeight(center.WorldX, center.WorldZ)
	if height <= SeaLevel || height > 210 {
		t.Fatalf("located village terrain height = %d", height)
	}

	chunkX := int32(floorDiv(center.WorldX, SectionSize))
	chunkZ := int32(floorDiv(center.WorldZ, SectionSize))
	generated := generator.VillageCentersNear(chunkX, chunkZ)
	matched := false
	for _, candidate := range generated {
		if candidate == center {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("located center %+v is not returned by village generation", center)
	}

	again, ok := generator.NearestVillage(0, 0, 8192)
	if !ok || again != center {
		t.Fatalf("village lookup is not deterministic: first=%+v second=%+v", center, again)
	}
}

func TestNearestBiomeFindsTheBiomeAtTheSearchOrigin(t *testing.T) {
	generator := NewOverworldGenerator(12345)
	const x, z = -137, 941
	target := generator.BiomeAt(x, z)

	gotX, gotZ, ok := generator.NearestBiome(x, z, target, 8192)
	if !ok {
		t.Fatalf("expected to locate origin biome %q", target)
	}
	if gotX != x || gotZ != z {
		t.Fatalf("located origin biome at %d,%d, want %d,%d", gotX, gotZ, x, z)
	}
}

func TestNearestBiomeFindsThreeDimensionalCaveBiomes(t *testing.T) {
	generator := NewOverworldGenerator(20260808)
	for _, target := range []string{"minecraft:lush_caves", "minecraft:dripstone_caves", "minecraft:deep_dark"} {
		x, z, ok := generator.NearestBiome(0, 0, target, 2048)
		if !ok {
			t.Errorf("expected to locate %s", target)
			continue
		}
		found := false
		surface := generator.sampleTerrain(x, z)
		for y := WorldMinY + 8; y <= minInt(56, surface.height-8); y += 12 {
			if generator.biomeAt3D(x, y, z, surface) == target {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("located %s at %d,%d but no sampled Y has that biome", target, x, z)
		}
	}
}

func TestGeneratedBiomeNamesAreUniqueAndUnnamespaced(t *testing.T) {
	seen := make(map[string]struct{})
	for _, name := range GeneratedBiomeNames() {
		if name == "" || strings.Contains(name, ":") {
			t.Errorf("invalid generated biome completion %q", name)
		}
		if _, duplicate := seen[name]; duplicate {
			t.Errorf("duplicate generated biome completion %q", name)
		}
		seen[name] = struct{}{}
	}
	if len(seen) == 0 {
		t.Fatal("generated biome completion list is empty")
	}
}
