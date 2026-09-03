package world

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestTorchBlockLightPropagatesInsideChunk(t *testing.T) {
	chunk := &coreworld.Chunk{X: 0, Z: 0}
	sectionIndex := (64 - coreworld.WorldMinY) / 16
	chunk.Sections[sectionIndex] = coreworld.NewSection()
	chunk.Sections[sectionIndex].Set(8, 0, 8, coreworld.Block{Namespace: "minecraft", Name: "torch"})
	levels := computeBlockLightRegion(localBlockLightRegion(chunk))

	if got := levels[blockLightRegionIndex(24, 64, 24)]; got != 14 {
		t.Fatalf("torch level = %d, want 14", got)
	}
	if got := levels[blockLightRegionIndex(25, 64, 24)]; got != 13 {
		t.Fatalf("adjacent level = %d, want 13", got)
	}
}

func TestTorchBlockLightCrossesChunkBorder(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(15, 64, 8, coreworld.Block{Namespace: "minecraft", Name: "torch"})
	target := world.Chunk(1, 0)
	levels := computeBlockLightRegion(worldBlockLightRegion(world, target))

	if got := levels[blockLightRegionIndex(16, 64, 24)]; got != 13 {
		t.Fatalf("light across chunk border = %d, want 13", got)
	}
}

func TestBlockLightMasksContainTorchSection(t *testing.T) {
	chunk := &coreworld.Chunk{}
	sectionIndex := (64 - coreworld.WorldMinY) / 16
	chunk.Sections[sectionIndex] = coreworld.NewSection()
	chunk.Sections[sectionIndex].Set(8, 0, 8, coreworld.Block{Namespace: "minecraft", Name: "wall_torch"})
	mask, empty, arrays := buildBlockLight(localBlockLightRegion(chunk))
	defer releaseSkyLightPages(arrays)
	bit := int64(1) << (sectionIndex + 1)
	if mask&bit == 0 || empty&bit != 0 || len(arrays) == 0 {
		t.Fatalf("block light masks omit torch section: mask=%026b empty=%026b arrays=%d", mask, empty, len(arrays))
	}
}

func TestBlockLightChangeDetectsEmitterPlacementAndRemoval(t *testing.T) {
	torch := coreworld.Block{Namespace: "minecraft", Name: "torch"}
	if !BlockLightChanged(coreworld.Air, torch) || !BlockLightChanged(torch, coreworld.Air) {
		t.Fatal("torch placement/removal did not invalidate block light")
	}
	if BlockLightChanged(torch, torch) {
		t.Fatal("unchanged torch invalidated block light")
	}
}
