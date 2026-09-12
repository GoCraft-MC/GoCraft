package server

import (
	"testing"

	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestBookBearingLecternOpensInsteadOfEjecting(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	pos := spatial.BlockPos{X: 1, Y: 64, Z: 0}
	lectern := coreworld.Block{Namespace: "minecraft", Name: "lectern", Properties: map[string]string{"has_book": "true"}}
	s.world.SetBlock(1, 64, 0, lectern)
	s.world.SetContainerItems(1, 64, 0, "minecraft:lectern", []coreworld.ContainerItem{{
		Slot: 0, ItemID: "minecraft:written_book", Count: 1,
	}})
	if s.applyBedrockBlockActivation(p, pos, lectern) {
		t.Fatal("book-bearing lectern consumed activation before its screen could open")
	}
	if got := coreworld.LecternBook(s.world.GetBlockEntity(1, 64, 0)); got != "minecraft:written_book" {
		t.Fatalf("lectern book was ejected: %q", got)
	}
}
