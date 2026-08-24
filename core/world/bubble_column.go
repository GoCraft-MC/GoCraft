package world

// UpdateBubbleColumnsAround updates the water column affected by a block
// change. Soul sand creates upward columns, magma creates downward columns,
// and removing or interrupting their support restores source water.
func (w *World) UpdateBubbleColumnsAround(x, y, z int) []BlockChange {
	seen := make(map[[3]int]struct{})
	changes := make([]BlockChange, 0)
	for _, supportY := range []int{y, y - 1} {
		for _, change := range w.updateBubbleColumnFromSupport(x, supportY, z) {
			key := [3]int{change.X, change.Y, change.Z}
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			seen[key] = struct{}{}
			changes = append(changes, change)
		}
	}
	return changes
}

func (w *World) updateBubbleColumnFromSupport(x, supportY, z int) []BlockChange {
	if supportY < WorldMinY || supportY >= WorldMaxY {
		return nil
	}
	support, loaded := w.blockIfLoaded(x, supportY, z)
	if !loaded {
		return nil
	}
	dragDown, active := false, false
	switch support.ResourceLocation() {
	case "minecraft:magma_block":
		dragDown, active = true, true
	case "minecraft:soul_sand":
		active = true
	case "minecraft:bubble_column":
		dragDown, active = support.Properties["drag"] == "true", true
	}

	changes := make([]BlockChange, 0)
	for columnY := supportY + 1; columnY <= WorldMaxY; columnY++ {
		current, columnLoaded := w.blockIfLoaded(x, columnY, z)
		if !columnLoaded {
			break
		}
		name := current.ResourceLocation()
		if active {
			if name != "minecraft:bubble_column" && (name != "minecraft:water" || FluidLevel(current) != 0) {
				break
			}
			replacement := Block{Namespace: "minecraft", Name: "bubble_column", Properties: map[string]string{"drag": boolString(dragDown)}}
			if current.Key() == replacement.Key() {
				continue
			}
			w.SetBlock(x, columnY, z, replacement)
			changes = append(changes, BlockChange{X: x, Y: columnY, Z: z, Block: replacement})
			continue
		}
		if name != "minecraft:bubble_column" {
			break
		}
		replacement := MakeFluid("minecraft:water", 0)
		w.SetBlock(x, columnY, z, replacement)
		changes = append(changes, BlockChange{X: x, Y: columnY, Z: z, Block: replacement})
	}
	return changes
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
