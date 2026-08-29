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

func TestSetBlockEntityNotifiesObserverWithOwnedData(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	var observed BlockEntity
	w.SetBlockEntityObserver(func(entity BlockEntity) { observed = entity })
	data := []byte{10, 0}
	w.SetBlockEntity(4, 65, 2, "minecraft:sign", data)
	data[0] = 0
	if observed.Type != "minecraft:sign" || observed.X != 4 || observed.Y != 65 || observed.Z != 2 {
		t.Fatalf("observer snapshot = %+v", observed)
	}
	if len(observed.Data) != 2 || observed.Data[0] != 10 {
		t.Fatalf("observer data aliases caller: %v", observed.Data)
	}
}
