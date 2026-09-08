package world

import "testing"

func TestBlockIfLoadedDoesNotGenerateTerrain(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	if _, loaded := w.BlockIfLoaded(16, 64, -1); loaded {
		t.Fatal("missing chunk reported as loaded")
	}
	if w.IsChunkLoaded(1, -1) {
		t.Fatal("block lookup generated terrain")
	}
}

func TestBlockIfLoadedMatchesGetBlock(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	for _, position := range [][3]int{{0, 64, 0}, {-1, -64, -17}, {16, 319, 31}} {
		x, y, z := position[0], position[1], position[2]
		w.SetBlock(x, y, z, Block{Namespace: "minecraft", Name: "stone"})
		block, loaded := w.BlockIfLoaded(x, y, z)
		if !loaded || block.Key() != w.GetBlock(x, y, z).Key() {
			t.Fatalf("block at %v = %+v, loaded=%v", position, block, loaded)
		}
	}
	for _, y := range []int{WorldMinY - 1, 65, WorldMaxY + 1} {
		if block, loaded := w.BlockIfLoaded(0, y, 0); !loaded || !block.IsAir() {
			t.Fatalf("empty block at y=%d = %+v, loaded=%v", y, block, loaded)
		}
	}
}
