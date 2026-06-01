package world

// NetherPortalInterior returns the six portal-block changes needed to fill a
// standard 4x5 obsidian frame containing the clicked obsidian block. Frames in
// either horizontal axis are accepted and their four corner blocks are
// optional, matching vanilla portal construction.
func NetherPortalInterior(w *World, clickedX, clickedY, clickedZ int) ([]BlockChange, bool) {
	if w == nil || w.GetBlock(clickedX, clickedY, clickedZ).ResourceLocation() != "minecraft:obsidian" {
		return nil, false
	}
	for _, axis := range []string{"x", "z"} {
		for horizontalOffset := -3; horizontalOffset <= 0; horizontalOffset++ {
			for verticalOffset := -4; verticalOffset <= 0; verticalOffset++ {
				baseX, bottom, baseZ := clickedX, clickedY+verticalOffset, clickedZ
				if axis == "x" {
					baseX += horizontalOffset
				} else {
					baseZ += horizontalOffset
				}
				if !isNetherPortalFrame(w, baseX, bottom, baseZ, axis) {
					continue
				}
				changes := make([]BlockChange, 0, 6)
				for horizontal := 1; horizontal <= 2; horizontal++ {
					for vertical := 1; vertical <= 3; vertical++ {
						x, z := baseX, baseZ
						if axis == "x" {
							x += horizontal
						} else {
							z += horizontal
						}
						changes = append(changes, BlockChange{X: x, Y: bottom + vertical, Z: z, Block: Block{
							Namespace: "minecraft", Name: "nether_portal", Properties: map[string]string{"axis": axis},
						}})
					}
				}
				return changes, true
			}
		}
	}
	return nil, false
}

func isNetherPortalFrame(w *World, baseX, bottom, baseZ int, axis string) bool {
	for horizontal := 0; horizontal < 4; horizontal++ {
		for vertical := 0; vertical < 5; vertical++ {
			border := ((horizontal == 0 || horizontal == 3) && vertical >= 1 && vertical <= 3) ||
				((vertical == 0 || vertical == 4) && horizontal >= 1 && horizontal <= 2)
			interior := horizontal >= 1 && horizontal <= 2 && vertical >= 1 && vertical <= 3
			x, z := baseX, baseZ
			if axis == "x" {
				x += horizontal
			} else {
				z += horizontal
			}
			block := w.GetBlock(x, bottom+vertical, z)
			if border && block.ResourceLocation() != "minecraft:obsidian" {
				return false
			}
			if interior && !netherPortalReplaceable(block.ResourceLocation()) {
				return false
			}
		}
	}
	return true
}

func netherPortalReplaceable(name string) bool {
	switch name {
	case "", "minecraft:air", "minecraft:cave_air", "minecraft:void_air", "minecraft:fire",
		"minecraft:short_grass", "minecraft:grass", "minecraft:fern", "minecraft:tall_grass",
		"minecraft:large_fern", "minecraft:dead_bush", "minecraft:snow", "minecraft:vine",
		"minecraft:water", "minecraft:lava", "minecraft:dandelion", "minecraft:poppy",
		"minecraft:blue_orchid", "minecraft:allium", "minecraft:azure_bluet",
		"minecraft:red_tulip", "minecraft:orange_tulip", "minecraft:white_tulip",
		"minecraft:pink_tulip", "minecraft:oxeye_daisy", "minecraft:cornflower",
		"minecraft:lily_of_the_valley":
		return true
	default:
		return false
	}
}
