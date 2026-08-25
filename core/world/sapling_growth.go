package world

import "strings"

type treePlacement struct {
	x, y, z int
	block   Block
}

// growSaplingTree builds a bounded vanilla-shaped tree without loading new
// chunks. The generated shape is shared by Java and Bedrock bonemeal use.
func (w *World) growSaplingTree(x, y, z int, sapling string, seed uint64) []BlockChange {
	wood := strings.TrimSuffix(strings.TrimPrefix(sapling, "minecraft:"), "_sapling")
	if sapling == "minecraft:mangrove_propagule" {
		wood = "mangrove"
	}
	if wood == "" || wood == "bamboo" {
		return nil
	}
	log := Block{Namespace: "minecraft", Name: wood + "_log", Properties: map[string]string{"axis": "y"}}
	leaves := Block{Namespace: "minecraft", Name: wood + "_leaves", Properties: map[string]string{
		"distance": "1", "persistent": "false", "waterlogged": "false",
	}}
	height := 4 + int(seed%3)
	placements := make(map[[3]int]Block)
	place := func(px, py, pz int, block Block, overwrite bool) {
		key := [3]int{px, py, pz}
		if _, exists := placements[key]; !exists || overwrite {
			placements[key] = block
		}
	}

	trunkX, trunkZ := x, z
	leanHeight, leanSteps := height, 0
	leanDirection := horizontalCropDirections[0]
	if wood == "acacia" {
		leanDirection = horizontalCropDirections[cropRandom(seed, x, y, z, 0xa11, 4)]
		leanHeight = height - cropRandom(seed, x, y, z, 0xa12, 4) - 1
		leanSteps = 3 - cropRandom(seed, x, y, z, 0xa13, 3)
	}
	for dy := 0; dy < height; dy++ {
		if wood == "acacia" && dy >= leanHeight && leanSteps > 0 {
			trunkX += leanDirection.dx
			trunkZ += leanDirection.dz
			leanSteps--
		}
		place(trunkX, y+dy, trunkZ, log, true)
	}
	topY := y + height - 1

	switch wood {
	case "spruce":
		for dy := -3; dy <= 1; dy++ {
			radius := 1
			if dy == -2 || dy == 0 {
				radius = 2
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if abs(dx)+abs(dz) <= radius+1 {
						place(trunkX+dx, topY+dy, trunkZ+dz, leaves, false)
					}
				}
			}
		}
	case "acacia":
		placeAcaciaFoliage(place, trunkX, topY+1, trunkZ, 1, leaves)
		branchDirection := horizontalCropDirections[cropRandom(seed, x, y, z, 0xa14, 4)]
		if branchDirection != leanDirection {
			branchY := max(1, leanHeight-cropRandom(seed, x, y, z, 0xa15, 2)-1)
			branchX, branchZ := x, z
			branchTop := y + branchY
			steps := 1 + cropRandom(seed, x, y, z, 0xa16, 3)
			for step := 0; step < steps && branchY+step < height; step++ {
				branchX += branchDirection.dx
				branchZ += branchDirection.dz
				branchTop = y + branchY + step
				place(branchX, branchTop, branchZ, log, true)
			}
			placeAcaciaFoliage(place, branchX, branchTop+1, branchZ, 0, leaves)
		}
	case "cherry":
		for _, branch := range horizontalCropDirections {
			place(x+branch.dx, topY-1, z+branch.dz, log, true)
			place(x+branch.dx*2, topY, z+branch.dz*2, log, true)
		}
		for dy := -2; dy <= 2; dy++ {
			radius := 4 - abs(dy)
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if abs(dx)+abs(dz) <= radius+1 {
						place(x+dx, topY+dy, z+dz, leaves, false)
					}
				}
			}
		}
	default:
		for dy := -2; dy <= 1; dy++ {
			radius := 2
			if dy == 1 {
				radius = 1
			}
			for dx := -radius; dx <= radius; dx++ {
				for dz := -radius; dz <= radius; dz++ {
					if radius == 2 && abs(dx) == 2 && abs(dz) == 2 {
						continue
					}
					place(x+dx, topY+dy, z+dz, leaves, false)
				}
			}
		}
	}

	for position := range placements {
		existing, loaded := w.blockIfLoaded(position[0], position[1], position[2])
		if !loaded || (!existing.IsAir() && !strings.HasSuffix(existing.ResourceLocation(), "_leaves") &&
			existing.ResourceLocation() != sapling) {
			return nil
		}
	}
	changes := make([]BlockChange, 0, len(placements))
	for position, block := range placements {
		w.SetBlock(position[0], position[1], position[2], block)
		changes = append(changes, BlockChange{X: position[0], Y: position[1], Z: position[2], Block: block})
	}
	return changes
}

func placeAcaciaFoliage(place func(int, int, int, Block, bool), x, y, z, nodeRadius int, leaves Block) {
	for dy, radius := range map[int]int{-2: 1, -1: 2 + nodeRadius, 0: 1 + nodeRadius} {
		for dx := -radius; dx <= radius; dx++ {
			for dz := -radius; dz <= radius; dz++ {
				ax, az := abs(dx), abs(dz)
				invalid := ax == radius && az == radius && radius > 0
				if dy == 0 {
					invalid = (ax > 1 || az > 1) && ax != 0 && az != 0
				}
				if !invalid {
					place(x+dx, y+dy, z+dz, leaves, false)
				}
			}
		}
	}
}
