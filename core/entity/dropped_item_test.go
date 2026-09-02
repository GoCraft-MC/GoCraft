package entity

import (
	"testing"

	"GoCraft/core/player"
)

func TestDroppedItemPreservesCompleteStack(t *testing.T) {
	want := player.ItemStack{ItemID: "minecraft:potion", Count: 3, Enchantments: "minecraft:vanishing_curse=1"}
	if err := want.SetComponent("potion_contents", map[string]string{"potion": "minecraft:poison"}); err != nil {
		t.Fatal(err)
	}
	entity := New(1, [16]byte{1}, TypeItem, 0, 64, 0)
	entity.SetDroppedItem(want)
	if got := entity.DroppedItem(); got != want {
		t.Fatalf("dropped stack = %#v, want %#v", got, want)
	}
}
