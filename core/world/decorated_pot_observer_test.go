package world

import "testing"

func TestDecoratedPotDecorationMutationNotifiesObserver(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	var observed BlockEntity
	w.SetBlockEntityObserver(func(entity BlockEntity) { observed = entity })
	want := [4]string{"minecraft:angler_pottery_sherd", "minecraft:brick", "minecraft:flow_pottery_sherd", "minecraft:brick"}
	w.SetDecoratedPotDecorations(3, 64, -2, want)
	if observed.X != 3 || observed.Y != 64 || observed.Z != -2 || observed.Type != "minecraft:decorated_pot" {
		t.Fatalf("unexpected block entity observer snapshot: %+v", observed)
	}
	if observed.PotDecorations != want {
		t.Fatalf("observed decorations = %#v, want %#v", observed.PotDecorations, want)
	}
}
