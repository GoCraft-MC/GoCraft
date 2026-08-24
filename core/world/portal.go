package world

const (
	minNetherPortalInteriorWidth  = 2
	maxNetherPortalInteriorWidth  = 21
	minNetherPortalInteriorHeight = 3
	maxNetherPortalInteriorHeight = 21
)

// NetherPortalInterior finds a valid vanilla-sized Nether portal frame that
// contains the clicked obsidian block and returns all interior portal blocks.
// Vanilla portals may have an interior from 2x3 up to 21x21 blocks, and their
// four corner blocks are optional. Both X- and Z-oriented frames are accepted.
func NetherPortalInterior(w *World, clickedX, clickedY, clickedZ int) ([]BlockChange, bool) {
	if w == nil || w.GetBlock(clickedX, clickedY, clickedZ).ResourceLocation() != "minecraft:obsidian" {
		return nil, false
	}

	for _, axis := range []string{"x", "z"} {
		for interiorWidth := minNetherPortalInteriorWidth; interiorWidth <= maxNetherPortalInteriorWidth; interiorWidth++ {
			outerWidth := interiorWidth + 2
			for interiorHeight := minNetherPortalInteriorHeight; interiorHeight <= maxNetherPortalInteriorHeight; interiorHeight++ {
				outerHeight := interiorHeight + 2

				// The clicked obsidian may be anywhere on a required edge. Try every
				// possible frame origin that could contain that clicked edge block.
				for horizontalOffset := -(outerWidth - 1); horizontalOffset <= 0; horizontalOffset++ {
					for verticalOffset := -(outerHeight - 1); verticalOffset <= 0; verticalOffset++ {
						baseX, bottom, baseZ := clickedX, clickedY+verticalOffset, clickedZ
						if axis == "x" {
							baseX += horizontalOffset
						} else {
							baseZ += horizontalOffset
						}

						clickedHorizontal := -horizontalOffset
						clickedVertical := -verticalOffset
						if !netherPortalRequiredEdge(clickedHorizontal, clickedVertical, outerWidth, outerHeight) {
							continue
						}
						if !isNetherPortalFrame(w, baseX, bottom, baseZ, axis, outerWidth, outerHeight) {
							continue
						}

						changes := make([]BlockChange, 0, interiorWidth*interiorHeight)
						for horizontal := 1; horizontal < outerWidth-1; horizontal++ {
							for vertical := 1; vertical < outerHeight-1; vertical++ {
								x, z := baseX, baseZ
								if axis == "x" {
									x += horizontal
								} else {
									z += horizontal
								}
								changes = append(changes, BlockChange{
									X: x, Y: bottom + vertical, Z: z,
									Block: Block{Namespace: "minecraft", Name: "nether_portal", Properties: map[string]string{"axis": axis}},
								})
							}
						}
						return changes, true
					}
				}
			}
		}
	}
	return nil, false
}

// netherPortalRequiredEdge reports whether a location in an outer frame is a
// mandatory obsidian block. Corners deliberately return false: vanilla permits
// all four corners to be omitted.
func netherPortalRequiredEdge(horizontal, vertical, outerWidth, outerHeight int) bool {
	if horizontal < 0 || horizontal >= outerWidth || vertical < 0 || vertical >= outerHeight {
		return false
	}
	if (horizontal == 0 || horizontal == outerWidth-1) && vertical >= 1 && vertical <= outerHeight-2 {
		return true
	}
	return (vertical == 0 || vertical == outerHeight-1) && horizontal >= 1 && horizontal <= outerWidth-2
}

func isNetherPortalFrame(w *World, baseX, bottom, baseZ int, axis string, outerWidth, outerHeight int) bool {
	for horizontal := 0; horizontal < outerWidth; horizontal++ {
		for vertical := 0; vertical < outerHeight; vertical++ {
			requiredBorder := netherPortalRequiredEdge(horizontal, vertical, outerWidth, outerHeight)
			interior := horizontal >= 1 && horizontal <= outerWidth-2 && vertical >= 1 && vertical <= outerHeight-2
			if !requiredBorder && !interior {
				continue // optional corner
			}

			x, z := baseX, baseZ
			if axis == "x" {
				x += horizontal
			} else {
				z += horizontal
			}
			block := w.GetBlock(x, bottom+vertical, z)
			if requiredBorder && block.ResourceLocation() != "minecraft:obsidian" {
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
		"minecraft:nether_portal", "minecraft:short_grass", "minecraft:grass", "minecraft:fern", "minecraft:tall_grass",
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
