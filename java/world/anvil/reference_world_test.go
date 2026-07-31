package anvil

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	coreworld "GoCraft/core/world"
	javaworld "GoCraft/java/world"
)

// TestReferenceWorldCompatibility is an opt-in integration test for a real,
// unmodified Java 1.21.4 save. CI skips it unless GOCRAFT_REFERENCE_WORLD is set.
// The fixture is read-only: this test never constructs Storage.SaveChunk.
func TestReferenceWorldCompatibility(t *testing.T) {
	worldDir := os.Getenv("GOCRAFT_REFERENCE_WORLD")
	if worldDir == "" {
		t.Skip("set GOCRAFT_REFERENCE_WORLD to a Java 1.21.4 world directory")
	}

	metadata, err := LoadLevelMetadata(worldDir)
	if err != nil {
		t.Fatalf("level.dat: %v", err)
	}
	t.Logf("level.dat DataVersion=%d LevelName=%q seed=%d", metadata.DataVersion, metadata.LevelName, metadata.Seed)

	var chunks, decoded, biomeSections, heightmaps, blockEntities, propertyPalettes int
	roundTripChecked := false
	unknownStates := make(map[string]struct{})
	for cx := int32(0); cx < 32; cx++ {
		for cz := int32(0); cz < 32; cz++ {
			root, err := loadChunkFromRegion(worldDir, cx, cz)
			if err != nil {
				t.Fatalf("raw chunk (%d,%d): %v", cx, cz, err)
			}
			if root == nil {
				continue
			}
			chunks++
			if root["Heightmaps"].typ == tagCompound {
				heightmaps++
			}
			if entities := root["block_entities"]; entities.typ == tagList {
				blockEntities += len(entities.listV)
			}
			for _, sectionTag := range root["sections"].List() {
				if sectionTag.typ != tagCompound {
					continue
				}
				section := sectionTag.compound
				if section["biomes"].Get("palette").typ == tagList {
					biomeSections++
				}
				for _, paletteEntry := range section["block_states"].Get("palette").List() {
					if paletteEntry.Get("Properties").typ == tagCompound {
						propertyPalettes++
					}
				}
			}
			chunk, err := chunkFromNBT(root, cx, cz)
			if err != nil {
				t.Fatalf("decode chunk (%d,%d): %v", cx, cz, err)
			}
			if chunk != nil {
				decoded++
				for _, entity := range chunk.BlockEntities {
					if _, ok := javaworld.BlockEntityTypeID(entity.Type); !ok {
						t.Fatalf("chunk (%d,%d) block entity type %q has no protocol ID", cx, cz, entity.Type)
					}
					if len(entity.Data) == 0 || entity.Data[0] != byte(tagCompound) {
						t.Fatalf("chunk (%d,%d) block entity %q has invalid network NBT", cx, cz, entity.Type)
					}
				}
				for _, section := range chunk.Sections {
					if section == nil {
						continue
					}
					for _, material := range section.BlockPalette() {
						if !javaworld.HasExactState(material) {
							unknownStates[material.Key()] = struct{}{}
						}
					}
				}
				if !roundTripChecked && len(root["block_entities"].List()) > 0 {
					mergedBytes := encodeChunkNBTWithBase(chunk, root)
					merged, mergeErr := ReadRootCompound(bytes.NewReader(mergedBytes))
					if mergeErr != nil {
						t.Fatalf("round-trip NBT chunk (%d,%d): %v", cx, cz, mergeErr)
					}
					if got, want := len(merged["block_entities"].List()), len(root["block_entities"].List()); got != want {
						t.Fatalf("round-trip block entities=%d, want %d", got, want)
					}
					if merged["structures"].typ != root["structures"].typ {
						t.Fatalf("round-trip lost structures tag")
					}
					if !reflect.DeepEqual(merged["Heightmaps"], root["Heightmaps"]) {
						t.Fatalf("round-trip changed unchanged heightmaps")
					}
					redecoded, decodeErr := chunkFromNBT(merged, cx, cz)
					if decodeErr != nil {
						t.Fatalf("round-trip decode chunk (%d,%d): %v", cx, cz, decodeErr)
					}
					assertChunksEquivalent(t, chunk, redecoded)
					roundTripChecked = true
				}
			}
		}
	}
	if chunks == 0 || decoded == 0 {
		t.Fatalf("reference contains raw=%d decoded=%d chunks", chunks, decoded)
	}
	if len(unknownStates) != 0 {
		t.Fatalf("reference contains %d block states absent from protocol-769 report: %v", len(unknownStates), unknownStates)
	}
	if !roundTripChecked {
		t.Fatal("reference contains no block-entity chunk for round-trip coverage")
	}
	if biomeSections == 0 || heightmaps == 0 || propertyPalettes == 0 {
		t.Fatalf("missing reference coverage: biomeSections=%d heightmaps=%d propertyPalettes=%d", biomeSections, heightmaps, propertyPalettes)
	}
	t.Logf("chunks raw=%d decoded=%d biomeSections=%d heightmapped=%d blockEntities=%d propertyPalettes=%d", chunks, decoded, biomeSections, heightmaps, blockEntities, propertyPalettes)
}

func assertChunksEquivalent(t *testing.T, first, second *coreworld.Chunk) {
	t.Helper()
	if len(first.BlockEntities) != len(second.BlockEntities) {
		t.Fatalf("block entity count=%d, want %d", len(second.BlockEntities), len(first.BlockEntities))
	}
	for index := range first.BlockEntities {
		got, want := second.BlockEntities[index], first.BlockEntities[index]
		if got.X != want.X || got.Y != want.Y || got.Z != want.Z || got.Type != want.Type || !bytes.Equal(got.Data, want.Data) {
			t.Fatalf("block entity %d differs after round trip: got (%d,%d,%d %s), want (%d,%d,%d %s)",
				index, got.X, got.Y, got.Z, got.Type, want.X, want.Y, want.Z, want.Type)
		}
	}
	for sectionIndex := 0; sectionIndex < coreworld.SectionCount; sectionIndex++ {
		firstSection, secondSection := first.Sections[sectionIndex], second.Sections[sectionIndex]
		if (firstSection == nil) != (secondSection == nil) {
			t.Fatalf("section %d nil mismatch", sectionIndex)
		}
		if firstSection == nil {
			continue
		}
		for y := 0; y < 16; y++ {
			for z := 0; z < 16; z++ {
				for x := 0; x < 16; x++ {
					if got, want := secondSection.At(x, y, z), firstSection.At(x, y, z); !got.Equal(want) {
						t.Fatalf("section %d block (%d,%d,%d)=%s, want %s", sectionIndex, x, y, z, got.Key(), want.Key())
					}
				}
			}
		}
		for y := 0; y < 4; y++ {
			for z := 0; z < 4; z++ {
				for x := 0; x < 4; x++ {
					if got, want := secondSection.BiomeAtCell(x, y, z), firstSection.BiomeAtCell(x, y, z); got != want {
						t.Fatalf("section %d biome (%d,%d,%d)=%s, want %s", sectionIndex, x, y, z, got, want)
					}
				}
			}
		}
	}
}
