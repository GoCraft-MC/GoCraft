package world

import (
	"testing"

	"GoCraft/core/spatial"
)

func TestEnsureSafeNetherArrivalBuildsPlatformWhenColumnHasNoOpening(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	for y := 7; y <= 118; y++ {
		w.SetBlock(4, y, 6, block("netherrack"))
	}

	arrival := w.EnsureSafeArrival(spatial.Vec3{X: 4.5, Y: 64, Z: 6.5}, 1)
	if arrival != (spatial.Vec3{X: 4.5, Y: 64, Z: 6.5}) {
		t.Fatalf("arrival = %+v", arrival)
	}
	if got := w.GetBlock(4, 63, 6).ResourceLocation(); got != "minecraft:obsidian" {
		t.Fatalf("landing support = %q", got)
	}
	if !w.GetBlock(4, 64, 6).IsAir() || !w.GetBlock(4, 65, 6).IsAir() {
		t.Fatal("landing chamber was not cleared")
	}
}

func TestEnsureSafeArrivalRejectsLavaAndMagma(t *testing.T) {
	w := New(&FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(2, 63, 2, block("magma_block"))
	w.SetBlock(2, 64, 2, MakeFluid("minecraft:lava", 0))
	arrival := w.EnsureSafeArrival(spatial.Vec3{X: 2.5, Y: 64, Z: 2.5}, 1)
	if int(arrival.Y) == 64 && (w.GetBlock(2, 64, 2).ResourceLocation() == "minecraft:lava" || w.GetBlock(2, 63, 2).ResourceLocation() == "minecraft:magma_block") {
		t.Fatalf("unsafe lava arrival was not repaired: %+v", arrival)
	}
}
