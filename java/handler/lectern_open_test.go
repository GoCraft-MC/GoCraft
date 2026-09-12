package handler

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestOpenLecternTracksBookAndPage(t *testing.T) {
	p := player.New([16]byte{7}, "reader", player.ClientEditionJava)
	book := coreworld.ContainerItem{Slot: 0, ItemID: "minecraft:written_book", Count: 1, Components: `{"minecraft:custom_name":"Guide"}`}
	pos := spatial.BlockPos{X: 4, Y: 70, Z: -3}
	if err := openLectern(p, nil, pos, coreworld.BlockEntity{Items: []coreworld.ContainerItem{book}, LecternPage: 3}); err != nil {
		t.Fatal(err)
	}
	if p.OpenContainerKind != "minecraft:lectern" || p.OpenContainerPos != pos || len(p.ContainerSlots) != 1 || p.ContainerSlots[0] != book.Stack() {
		t.Fatalf("open lectern state = kind %q pos %+v slots %+v", p.OpenContainerKind, p.OpenContainerPos, p.ContainerSlots)
	}
}
