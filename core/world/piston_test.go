package world

import "testing"

func TestPistonPushesAndStickyPistonPulls(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "sticky_piston", Properties: map[string]string{
		"facing": "east", "extended": "false",
	}})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	if changes := w.ApplyPistonPower(0, 64, 0, true); len(changes) == 0 {
		t.Fatal("piston did not extend")
	}
	if got := w.GetBlock(1, 64, 0).ResourceLocation(); got != "minecraft:piston_head" {
		t.Fatalf("front block = %q, want piston head", got)
	}
	if got := w.GetBlock(2, 64, 0).ResourceLocation(); got != "minecraft:stone" {
		t.Fatalf("pushed block = %q, want stone", got)
	}
	w.ApplyPistonPower(0, 64, 0, false)
	if got := w.GetBlock(1, 64, 0).ResourceLocation(); got != "minecraft:stone" {
		t.Fatalf("pulled block = %q, want stone", got)
	}
	if got := w.GetBlock(2, 64, 0); !got.IsAir() {
		t.Fatalf("old pushed position = %q, want air", got.ResourceLocation())
	}
}

func TestPistonRespectsTwelveBlockPushLimit(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "piston", Properties: map[string]string{"facing": "east", "extended": "false"}})
	for x := 1; x <= 13; x++ {
		w.SetBlock(x, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	}
	if changes := w.ApplyPistonPower(0, 64, 0, true); len(changes) != 0 {
		t.Fatalf("over-limit piston produced %d changes", len(changes))
	}
}
