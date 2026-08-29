package world

import (
	"bytes"
	"testing"

	coreworld "GoCraft/core/world"
)

func TestBlockEntityModelsRetainOutdoorSkylight(t *testing.T) {
	for _, name := range []string{"minecraft:chest", "minecraft:trapped_chest", "minecraft:ender_chest", "minecraft:barrel", "minecraft:decorated_pot"} {
		if !isSkyTransparent(name) {
			t.Errorf("%s is opaque to the synthetic skylight engine", name)
		}
	}

	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(1, 64, 1, coreworld.Block{Namespace: "minecraft", Name: "chest"})
	chunk := w.Chunk(0, 0)
	levels := computeSkyLightLevels(chunk)
	defer releaseSkyLightLevels(levels)
	if got := levels[skyCellIndex(1, 64, 1)]; got != 15 {
		t.Fatalf("outdoor chest skylight = %d, want 15", got)
	}
}

func TestDecoratedPotNetworkDataUsesCanonicalSherds(t *testing.T) {
	entity := coreworld.BlockEntity{
		Type: "minecraft:decorated_pot",
		PotDecorations: [4]string{
			"minecraft:angler_pottery_sherd", "minecraft:brick",
			"minecraft:flow_pottery_sherd", "minecraft:miner_pottery_sherd",
		},
	}
	data := BlockEntityNetworkData(entity)
	for _, decoration := range entity.PotDecorations {
		if !bytes.Contains(data, []byte(decoration)) {
			t.Fatalf("network NBT does not contain %q: %x", decoration, data)
		}
	}
}
