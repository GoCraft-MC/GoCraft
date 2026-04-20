package world

import (
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
