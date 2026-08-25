package world

import "testing"

func TestBoneMealGrowsSaplingIntoTree(t *testing.T) {
	world := New(&FlatGenerator{}, nil, false)
	defer world.Close()
	world.SetBlock(8, 63, 8, Block{Namespace: "minecraft", Name: "dirt"})
	world.SetBlock(8, 64, 8, Block{Namespace: "minecraft", Name: "oak_sapling", Properties: map[string]string{"stage": "1"}})
	changes, used := world.ApplyBoneMeal(8, 64, 8, 1)
	if !used || len(changes) == 0 {
		t.Fatalf("sapling bonemeal: used=%v changes=%d", used, len(changes))
	}
	if got := world.GetBlock(8, 64, 8).ResourceLocation(); got != "minecraft:oak_log" {
		t.Fatalf("sapling base became %q, want oak log", got)
	}
}

func TestAcaciaAndCherryTreesUseDistinctShapes(t *testing.T) {
	for _, test := range []struct {
		sapling, log, leaves string
		minimumWidth         int
		minimumLogColumns    int
	}{
		{sapling: "acacia_sapling", log: "minecraft:acacia_log", leaves: "minecraft:acacia_leaves", minimumWidth: 5, minimumLogColumns: 2},
		{sapling: "cherry_sapling", log: "minecraft:cherry_log", leaves: "minecraft:cherry_leaves", minimumWidth: 7, minimumLogColumns: 5},
	} {
		t.Run(test.sapling, func(t *testing.T) {
			world := New(&FlatGenerator{}, nil, false)
			defer world.Close()
			world.SetBlock(8, 63, 8, Block{Namespace: "minecraft", Name: "dirt"})
			world.SetBlock(8, 64, 8, Block{Namespace: "minecraft", Name: test.sapling, Properties: map[string]string{"stage": "1"}})
			if changes, used := world.ApplyBoneMeal(8, 64, 8, 7); !used || len(changes) == 0 {
				t.Fatalf("tree growth used=%v changes=%d", used, len(changes))
			}
			minX, maxX := 100, -100
			logColumns := make(map[[2]int]struct{})
			foundLog, foundLeaves := false, false
			for x := 0; x < 16; x++ {
				for y := 64; y < 80; y++ {
					for z := 0; z < 16; z++ {
						name := world.GetBlock(x, y, z).ResourceLocation()
						if name == test.log {
							foundLog = true
							logColumns[[2]int{x, z}] = struct{}{}
						}
						if name == test.leaves {
							foundLeaves = true
							minX, maxX = min(minX, x), max(maxX, x)
						}
					}
				}
			}
			if !foundLog || !foundLeaves || maxX-minX+1 < test.minimumWidth {
				t.Fatalf("shape log=%v leaves=%v width=%d, want width >= %d", foundLog, foundLeaves, maxX-minX+1, test.minimumWidth)
			}
			if len(logColumns) < test.minimumLogColumns {
				t.Fatalf("log columns = %d, want at least %d", len(logColumns), test.minimumLogColumns)
			}
		})
	}
}
