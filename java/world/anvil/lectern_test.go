package anvil

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestLecternPageStateRoundTrip(t *testing.T) {
	want := coreworld.BlockEntity{
		X: 2, Y: 64, Z: 3, Type: "minecraft:lectern", Data: []byte{10, 0},
		Items:       []coreworld.ContainerItem{{Slot: 0, ItemID: "minecraft:written_book", Count: 1}},
		LecternPage: 4, LecternPageCount: 10,
	}
	got := decodeBlockEntities(blockEntitiesTag([]coreworld.BlockEntity{want}))
	if len(got) != 1 || got[0].LecternPage != 4 || got[0].LecternPageCount != 10 {
		t.Fatalf("lectern state after NBT round trip = %+v", got)
	}
}
