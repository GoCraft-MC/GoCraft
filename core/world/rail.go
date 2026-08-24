package world

func IsRailBlock(name string) bool {
	return name == "minecraft:rail" || name == "minecraft:powered_rail" ||
		name == "minecraft:detector_rail" || name == "minecraft:activator_rail"
}

// UpdateRailShapesAround reconnects nearby rails after placement or removal.
func (w *World) UpdateRailShapesAround(x, y, z int) []BlockChange {
	positions := make(map[[3]int]struct{})
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				position := [3]int{x + dx, y + dy, z + dz}
				if IsRailBlock(w.GetBlock(position[0], position[1], position[2]).ResourceLocation()) {
					positions[position] = struct{}{}
				}
			}
		}
	}
	changes := make([]BlockChange, 0, len(positions))
	for position := range positions {
		rail := w.GetBlock(position[0], position[1], position[2])
		shape := w.railShapeAt(position[0], position[1], position[2], rail)
		if rail.Properties["shape"] == shape {
			continue
		}
		rail = copyWorldBlock(rail)
		rail.Properties["shape"] = shape
		w.setBlockNoPhysics(position[0], position[1], position[2], rail)
		changes = append(changes, BlockChange{X: position[0], Y: position[1], Z: position[2], Block: rail})
	}
	return changes
}

func (w *World) railShapeAt(x, y, z int, rail Block) string {
	connected := func(dx, dz int) (bool, int) {
		for _, dy := range []int{0, 1, -1} {
			if IsRailBlock(w.GetBlock(x+dx, y+dy, z+dz).ResourceLocation()) {
				return true, dy
			}
		}
		return false, 0
	}
	north, northY := connected(0, -1)
	south, southY := connected(0, 1)
	east, eastY := connected(1, 0)
	west, westY := connected(-1, 0)
	straightOnly := rail.ResourceLocation() != "minecraft:rail"
	if east && west && !north && !south {
		if eastY > 0 {
			return "ascending_east"
		}
		if westY > 0 {
			return "ascending_west"
		}
		return "east_west"
	}
	if north && south && !east && !west {
		if northY > 0 {
			return "ascending_north"
		}
		if southY > 0 {
			return "ascending_south"
		}
		return "north_south"
	}
	if !straightOnly {
		switch {
		case south && east:
			return "south_east"
		case south && west:
			return "south_west"
		case north && west:
			return "north_west"
		case north && east:
			return "north_east"
		}
	}
	if east || west {
		return "east_west"
	}
	return "north_south"
}
