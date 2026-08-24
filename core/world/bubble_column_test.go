package world

import "testing"

func TestSoulSandAndMagmaCreateBubbleColumns(t *testing.T) {
	for _, test := range []struct {
		name, drag string
	}{
		{name: "soul_sand", drag: "false"},
		{name: "magma_block", drag: "true"},
	} {
		t.Run(test.name, func(t *testing.T) {
			world := New(&FlatGenerator{}, nil, false)
			defer world.Close()
			world.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: test.name})
			for y := 41; y <= 44; y++ {
				world.SetBlock(0, y, 0, MakeFluid("minecraft:water", 0))
			}
			changes := world.UpdateBubbleColumnsAround(0, 40, 0)
			if len(changes) != 4 {
				t.Fatalf("bubble changes = %d, want 4", len(changes))
			}
			for y := 41; y <= 44; y++ {
				block := world.GetBlock(0, y, 0)
				if block.ResourceLocation() != "minecraft:bubble_column" || block.Properties["drag"] != test.drag {
					t.Fatalf("bubble at y=%d = %+v, want drag=%s", y, block, test.drag)
				}
			}
		})
	}
}

func TestRemovingBubbleSupportRestoresSourceWater(t *testing.T) {
	world := New(&FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(0, 40, 0, Block{Namespace: "minecraft", Name: "soul_sand"})
	for y := 41; y <= 43; y++ {
		world.SetBlock(0, y, 0, MakeFluid("minecraft:water", 0))
	}
	world.UpdateBubbleColumnsAround(0, 40, 0)
	world.SetBlock(0, 40, 0, Air)
	changes := world.UpdateBubbleColumnsAround(0, 40, 0)
	if len(changes) != 3 {
		t.Fatalf("restored water changes = %d, want 3", len(changes))
	}
	for y := 41; y <= 43; y++ {
		block := world.GetBlock(0, y, 0)
		if block.ResourceLocation() != "minecraft:water" || FluidLevel(block) != 0 {
			t.Fatalf("restored block at y=%d = %+v", y, block)
		}
	}
}
