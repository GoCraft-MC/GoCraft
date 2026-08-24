package world

import "testing"

func buildTestPortalFrame(w *World, baseX, bottom, baseZ int, axis string, interiorWidth, interiorHeight int) {
	outerWidth, outerHeight := interiorWidth+2, interiorHeight+2
	for horizontal := 0; horizontal < outerWidth; horizontal++ {
		for vertical := 0; vertical < outerHeight; vertical++ {
			if !netherPortalRequiredEdge(horizontal, vertical, outerWidth, outerHeight) {
				continue
			}
			x, z := baseX, baseZ
			if axis == "x" {
				x += horizontal
			} else {
				z += horizontal
			}
			w.SetBlock(x, bottom+vertical, z, Block{Namespace: "minecraft", Name: "obsidian"})
		}
	}
}

func TestNetherPortalInteriorAcceptsVanillaVariableSizeFrame(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	buildTestPortalFrame(w, 10, 64, 10, "x", 5, 6)
	changes, ok := NetherPortalInterior(w, 10, 66, 10)
	if !ok {
		t.Fatal("expected a valid 5x6 interior portal frame")
	}
	if len(changes) != 30 {
		t.Fatalf("portal changes = %d, want 30", len(changes))
	}
	for _, change := range changes {
		if change.Block.ResourceLocation() != "minecraft:nether_portal" || change.Block.Properties["axis"] != "x" {
			t.Fatalf("unexpected portal block: %s", change.Block.Key())
		}
	}
}

func TestNetherPortalInteriorAcceptsZAxisAndOptionalCorners(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	buildTestPortalFrame(w, -4, 70, 20, "z", 3, 4)
	changes, ok := NetherPortalInterior(w, -4, 72, 20)
	if !ok {
		t.Fatal("expected valid z-axis portal without corner obsidian")
	}
	if len(changes) != 12 {
		t.Fatalf("portal changes = %d, want 12", len(changes))
	}
	for _, change := range changes {
		if change.Block.Properties["axis"] != "z" {
			t.Fatalf("axis = %q, want z", change.Block.Properties["axis"])
		}
	}
}

func TestNetherPortalInteriorRejectsBrokenFrame(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	buildTestPortalFrame(w, 0, 64, 0, "x", 2, 3)
	w.SetBlock(3, 66, 0, Air) // remove one mandatory side block
	if _, ok := NetherPortalInterior(w, 0, 66, 0); ok {
		t.Fatal("broken frame was accepted")
	}
}
