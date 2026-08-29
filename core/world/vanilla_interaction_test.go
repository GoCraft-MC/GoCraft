package world

import "testing"

func TestDoorHingeUsesVanillaCursorSide(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	if got := DoorHinge(w, 0, 64, 0, "north", 0.75, 0.5); got != "right" {
		t.Fatalf("north door clicked on east half hinge = %q, want right", got)
	}
	if got := DoorHinge(w, 0, 64, 0, "north", 0.25, 0.5); got != "left" {
		t.Fatalf("north door clicked on west half hinge = %q, want left", got)
	}
	if got := DoorHinge(w, 0, 64, 0, "east", 0.5, 0.75); got != "right" {
		t.Fatalf("east door clicked on south half hinge = %q, want right", got)
	}
}

func TestDoorHingeFormsVanillaDoubleDoor(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	// For a north-facing placement, counter-clockwise/left is west.
	w.SetBlock(-1, 64, 0, Block{Namespace: "minecraft", Name: "oak_door", Properties: map[string]string{
		"facing": "north", "half": "lower", "hinge": "left", "open": "false", "powered": "false",
	}})
	if got := DoorHinge(w, 0, 64, 0, "north", 0.25, 0.5); got != "right" {
		t.Fatalf("door next to left-hand door hinge = %q, want right", got)
	}
}

func TestDoorHingeObstructionOverridesCursor(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	// North-facing left side is west. Vanilla chooses LEFT when that side is
	// more obstructed, even though this cursor position alone would choose RIGHT.
	w.SetBlock(-1, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(-1, 65, 0, Block{Namespace: "minecraft", Name: "stone"})
	if got := DoorHinge(w, 0, 64, 0, "north", 0.75, 0.5); got != "left" {
		t.Fatalf("obstructed-left hinge = %q, want left", got)
	}
}

func TestBreakingWallLeverSupportRemovesLever(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(0, 64, -1, Block{Namespace: "minecraft", Name: "lever", Properties: map[string]string{
		"face": "wall", "facing": "north", "powered": "false",
	}})
	w.SetBlock(0, 64, 0, Air)
	changes := w.BreakUnsupportedAttachmentsAround(0, 64, 0)
	if len(changes) != 1 {
		t.Fatalf("attachment changes = %d, want 1", len(changes))
	}
	if got := w.GetBlock(0, 64, -1); !got.IsAir() {
		t.Fatalf("lever after support break = %q, want air", got.ResourceLocation())
	}
}

func TestBreakingFloorButtonSupportRemovesButton(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	w.SetBlock(0, 63, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "stone_button", Properties: map[string]string{
		"face": "floor", "facing": "north", "powered": "false",
	}})
	w.SetBlock(0, 63, 0, Air)
	w.BreakUnsupportedAttachmentsAround(0, 63, 0)
	if got := w.GetBlock(0, 64, 0); !got.IsAir() {
		t.Fatalf("button after support break = %q, want air", got.ResourceLocation())
	}
}

func TestSolidBlockCarriesPowerToAdjacentPiston(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "redstone_block"})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(2, 64, 0, Block{Namespace: "minecraft", Name: "piston", Properties: map[string]string{
		"facing": "east", "extended": "false",
	}})
	result := w.Redstone.FlushUpdates()
	if got := w.Redstone.PowerAt(1, 64, 0); got == 0 {
		t.Fatal("solid block adjacent to redstone block did not become powered")
	}
	if got := w.Redstone.PowerAt(2, 64, 0); got == 0 {
		t.Fatal("piston did not receive power through solid block")
	}
	found := false
	for _, pos := range result.PoweredLoads {
		if pos == [3]int{2, 64, 0} {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("piston power transition was not reported as a powered load")
	}
}

func TestSolidBlocksDoNotRelayPowerIndefinitely(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()

	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "redstone_block"})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(2, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.Redstone.FlushUpdates()
	if got := w.Redstone.PowerAt(1, 64, 0); got == 0 {
		t.Fatal("first solid block should be powered")
	}
	if got := w.Redstone.PowerAt(2, 64, 0); got != 0 {
		t.Fatalf("power chained through two solid blocks: %d", got)
	}
}

func TestObserverSchedulesTwoTickPulse(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetPhysicsTime(100)
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "observer", Properties: map[string]string{"facing": "east", "powered": "false"}})
	// Ignore placement's own pending bookkeeping, then change the exact watched block.
	w.BlockPhysics.DrainDue(100)
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	if got := w.GetBlock(0, 64, 0).Properties["powered"]; got == "true" {
		t.Fatal("observer powered before scheduled detection tick")
	}
	if due := w.BlockPhysics.DrainDue(101); len(due) != 0 {
		t.Fatalf("observer fired early: %+v", due)
	}
	due := w.BlockPhysics.DrainDue(102)
	if len(due) != 1 || due[0].Kind != UpdateObserver {
		t.Fatalf("due=%+v, want one observer update", due)
	}
}

func TestObserverOnlyWatchesItsFront(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetPhysicsTime(10)
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "observer", Properties: map[string]string{"facing": "east", "powered": "false"}})
	w.BlockPhysics.DrainDue(10)
	w.SetBlock(0, 64, -1, Block{Namespace: "minecraft", Name: "stone"})
	if due := w.BlockPhysics.DrainDue(12); len(due) != 0 {
		t.Fatalf("side change triggered observer: %+v", due)
	}
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "fire"})
	if due := w.BlockPhysics.DrainDue(12); len(due) != 1 || due[0].Kind != UpdateObserver {
		t.Fatalf("front fire change did not trigger: %+v", due)
	}
}
