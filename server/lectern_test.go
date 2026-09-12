package server

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestLecternPageIntentUpdatesCanonicalPage(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	pos := spatial.BlockPos{X: 1, Y: 64, Z: 0}
	lectern := coreworld.Block{Namespace: "minecraft", Name: "lectern", Properties: map[string]string{
		"has_book": "true", "powered": "false",
	}}
	s.world.SetBlock(1, 64, 0, lectern)
	s.world.SetContainerItems(1, 64, 0, "minecraft:lectern", []coreworld.ContainerItem{{
		Slot: 0, ItemID: "minecraft:written_book", Count: 1,
	}})
	s.applyLecternPage(intent.LecternPageIntent{
		PlayerUUID: p.UUID, Dimension: p.Dimension, Position: pos, Page: 4, PageCount: 10,
	})
	if got := s.world.GetBlockEntity(1, 64, 0); got.LecternPage != 4 || got.LecternPageCount != 10 {
		t.Fatalf("absolute page update = %d/%d, want 4/10", got.LecternPage, got.LecternPageCount)
	}
	s.applyLecternPage(intent.LecternPageIntent{
		PlayerUUID: p.UUID, Dimension: p.Dimension, Position: pos, Page: -1, Relative: true,
	})
	if got := s.world.GetBlockEntity(1, 64, 0).LecternPage; got != 3 {
		t.Fatalf("relative page update = %d, want 3", got)
	}
}
