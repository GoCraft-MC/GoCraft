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
		direction := horizontalCropDirections[cropRandom(seed, x, y, z, 0xc11, 4)]
		branchCount := 2 + cropRandom(seed, x, y, z, 0xc12, 2)
		for branch := 0; branch < 2; branch++ {
			if branch == 1 {
				direction.dx, direction.dz = -direction.dx, -direction.dz
			}
			axis := "z"
			if direction.dx != 0 {
				axis = "x"
			}
			branchLog := Block{Namespace: "minecraft", Name: wood + "_log", Properties: map[string]string{"axis": axis}}
			branchY := topY - 1 - branch
			endX, endZ := x, z
			for step := 1; step <= 2; step++ {
				endX, endZ = x+direction.dx*step, z+direction.dz*step
				place(endX, branchY, endZ, branchLog, true)
			}
			endY := branchY + cropRandom(seed, endX, branchY, endZ, 0xc13, 2)
			if endY > branchY {
				place(endX, endY, endZ, log, true)
			}
			placeCherryFoliage(place, endX, endY+1, endZ, seed, leaves)
		}
		if branchCount == 3 {
			placeCherryFoliage(place, x, topY+1, z, seed^0xc14, leaves)
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

func placeCherryFoliage(place func(int, int, int, Block, bool), x, y, z int, seed uint64, leaves Block) {
	layers := [...]struct{ dy, radius int }{{2, 1}, {1, 2}, {0, 3}, {-1, 3}, {-2, 2}}
	for _, layer := range layers {
		for dx := -layer.radius; dx <= layer.radius; dx++ {
			for dz := -layer.radius; dz <= layer.radius; dz++ {
				corner := abs(dx) == layer.radius && abs(dz) == layer.radius
				if corner || layer.dy == -1 && cropRandom(seed, x+dx, y+layer.dy, z+dz, 0xc15, 5) == 0 {
					continue
				}
				place(x+dx, y+layer.dy, z+dz, leaves, false)
				perimeter := abs(dx) == layer.radius || abs(dz) == layer.radius
				if layer.dy == -1 && perimeter && cropRandom(seed, x+dx, y, z+dz, 0xc16, 4) == 0 {
					place(x+dx, y-2, z+dz, leaves, false)
					if cropRandom(seed, x+dx, y, z+dz, 0xc17, 3) == 0 {
						place(x+dx, y-3, z+dz, leaves, false)
					}
				}
			}
		}
	}
}
