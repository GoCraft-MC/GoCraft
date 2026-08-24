package world

import "testing"

func TestLeverWirePowersAndUnpowersLamp(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "lever", Properties: map[string]string{"powered": "true"}})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "redstone_wire", Properties: map[string]string{"power": "0"}})
	w.SetBlock(2, 64, 0, Block{Namespace: "minecraft", Name: "redstone_lamp", Properties: map[string]string{"lit": "false"}})

	w.Redstone.FlushUpdates()
	if got := w.GetBlock(1, 64, 0).Properties["power"]; got != "15" {
		t.Fatalf("wire power = %q, want 15", got)
	}
	if got := w.GetBlock(2, 64, 0).Properties["lit"]; got != "true" {
		t.Fatalf("lamp lit = %q, want true", got)
	}

	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "lever", Properties: map[string]string{"powered": "false"}})
	w.Redstone.FlushUpdates()
	if got := w.GetBlock(1, 64, 0).Properties["power"]; got != "0" {
		t.Fatalf("wire power after lever off = %q, want 0", got)
	}
	if got := w.GetBlock(2, 64, 0).Properties["lit"]; got != "false" {
		t.Fatalf("lamp lit after lever off = %q, want false", got)
	}
}

func TestEveryButtonAndPressurePlateIsRecognisedAsSource(t *testing.T) {
	for _, name := range []string{
		"minecraft:polished_blackstone_button", "minecraft:pale_oak_button",
		"minecraft:birch_pressure_plate", "minecraft:heavy_weighted_pressure_plate",
	} {
		if !IsRedstoneSource(name) {
			t.Errorf("%s was not recognised as a redstone source", name)
		}
	}
}

func TestRedstoneTorchInvertsPowerConductedThroughSupport(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "lever", Properties: map[string]string{"powered": "true"}})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "redstone_wire", Properties: map[string]string{"power": "0"}})
	w.SetBlock(2, 64, 0, Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(2, 65, 0, Block{Namespace: "minecraft", Name: "redstone_torch", Properties: map[string]string{"lit": "true"}})
	w.SetBlock(3, 65, 0, Block{Namespace: "minecraft", Name: "redstone_lamp", Properties: map[string]string{"lit": "false"}})

	w.Redstone.FlushUpdates()
	if got := w.GetBlock(2, 65, 0).Properties["lit"]; got != "false" {
		t.Fatalf("powered support left torch lit = %q", got)
	}
	if got := w.GetBlock(3, 65, 0).Properties["lit"]; got != "false" {
		t.Fatalf("inverted torch powered lamp = %q", got)
	}

	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "lever", Properties: map[string]string{"powered": "false"}})
	w.Redstone.FlushUpdates()
	if got := w.GetBlock(2, 65, 0).Properties["lit"]; got != "true" {
		t.Fatalf("unpowered support left torch unlit = %q", got)
	}
	if got := w.GetBlock(3, 65, 0).Properties["lit"]; got != "true" {
		t.Fatalf("torch did not activate lamp = %q", got)
	}
}

func TestRepeaterReadsRearAndPowersOnlyItsFront(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 1, Block{Namespace: "minecraft", Name: "redstone_block"})
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "repeater", Properties: map[string]string{
		"facing": "north", "delay": "1", "locked": "false", "powered": "false",
	}})
	w.SetBlock(0, 64, -1, Block{Namespace: "minecraft", Name: "redstone_lamp", Properties: map[string]string{"lit": "false"}})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "redstone_lamp", Properties: map[string]string{"lit": "false"}})

	w.Redstone.FlushUpdates()
	if got := w.GetBlock(0, 64, 0).Properties["powered"]; got != "true" {
		t.Fatalf("repeater powered = %q, want true", got)
	}
	if got := w.GetBlock(0, 64, -1).Properties["lit"]; got != "true" {
		t.Fatalf("front lamp lit = %q, want true", got)
	}
	if got := w.GetBlock(1, 64, 0).Properties["lit"]; got != "false" {
		t.Fatalf("side lamp lit = %q, want false", got)
	}
}

func TestComparatorSubtractsSideSignal(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(0, 64, 1, Block{Namespace: "minecraft", Name: "redstone_block"})
	w.SetBlock(3, 64, 0, Block{Namespace: "minecraft", Name: "redstone_block"})
	w.SetBlock(2, 64, 0, Block{Namespace: "minecraft", Name: "redstone_wire", Properties: map[string]string{"power": "0"}})
	w.SetBlock(1, 64, 0, Block{Namespace: "minecraft", Name: "redstone_wire", Properties: map[string]string{"power": "0"}})
	w.SetBlock(0, 64, 0, Block{Namespace: "minecraft", Name: "comparator", Properties: map[string]string{
		"facing": "north", "mode": "subtract", "powered": "false",
	}})

	w.Redstone.FlushUpdates()
	if got := w.Redstone.PowerAt(0, 64, 0); got != 1 {
		t.Fatalf("comparator output = %d, want 1", got)
	}
	if got := w.GetBlock(0, 64, 0).Properties["powered"]; got != "true" {
		t.Fatalf("comparator powered = %q, want true", got)
	}
}
