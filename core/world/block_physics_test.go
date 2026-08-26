package world

import "testing"

func TestScheduledPhysicsUsesMonotonicWorldAge(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetWorldTime(0)
	w.SetPhysicsTime(48000)
	w.SetBlock(0, 65, 0, MakeFluid("minecraft:water", 0))

	if due := w.BlockPhysics.DrainDue(48004); len(due) != 0 {
		t.Fatalf("fluid update fired early after day rollover: %+v", due)
	}
	if due := w.BlockPhysics.DrainDue(48005); len(due) != 1 || due[0].Kind != UpdateFluid {
		t.Fatalf("fluid update at tick 48005 = %+v", due)
	}
}
