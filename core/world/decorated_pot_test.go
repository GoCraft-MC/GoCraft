package world

import "testing"

func TestDecoratedPotDecorationsSurviveContainerUpdates(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "decorated_pot", Properties: map[string]string{"facing": "north", "cracked": "false", "waterlogged": "false"}})
	want := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:skull_pottery_sherd", "minecraft:heart_pottery_sherd"}
	w.SetDecoratedPotDecorations(0, 64, 0, want)
	w.SetContainerItems(0, 64, 0, "minecraft:decorated_pot", []ContainerItem{{Slot: 0, ItemID: "minecraft:diamond", Count: 3}})
	if got := w.DecoratedPotDecorations(0, 64, 0); got != want {
		t.Fatalf("decorations = %#v, want %#v", got, want)
	}
}
