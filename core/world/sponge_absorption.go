package world

// Sponge absorption follows vanilla: a freshly placed dry sponge removes water
// reachable within a taxicab distance of six, up to 64 blocks, and turns into a
// wet sponge if it absorbed anything. Waterlogged blocks are drained in place;
// full water blocks become air.
const (
	spongeMaxDistance = 6
	spongeMaxAbsorbed = 64
)

// absorbWaterAround runs the breadth-first absorption from a placed sponge and
// converts it to a wet sponge when at least one water block is removed. It uses
// SetBlock for each drained block so adapters observe the changes normally.
func (w *World) absorbWaterAround(x, y, z int) bool {
	type node struct {
		x, y, z, dist int
	}
	start := [3]int{x, y, z}
	visited := map[[3]int]bool{start: true}
	queue := []node{{x, y, z, 0}}
	drained := make([][3]int, 0, spongeMaxAbsorbed)

	for len(queue) > 0 && len(drained) < spongeMaxAbsorbed {
		current := queue[0]
		queue = queue[1:]
		if current.dist >= spongeMaxDistance {
			continue
		}
		for _, neighbor := range neighbors6(current.x, current.y, current.z) {
			key := [3]int{neighbor[0], neighbor[1], neighbor[2]}
			if visited[key] {
				continue
			}
			visited[key] = true
			block, loaded := w.blockIfLoaded(neighbor[0], neighbor[1], neighbor[2])
			if !loaded || !spongeAbsorbable(block) {
				continue
			}
			drained = append(drained, key)
			queue = append(queue, node{neighbor[0], neighbor[1], neighbor[2], current.dist + 1})
			if len(drained) >= spongeMaxAbsorbed {
				break
			}
		}
	}

	if len(drained) == 0 {
		return false
	}
	for _, position := range drained {
		block, _ := w.blockIfLoaded(position[0], position[1], position[2])
		if block.Properties["waterlogged"] == "true" {
			drainedBlock := copyWorldBlock(block)
			drainedBlock.Properties["waterlogged"] = "false"
			w.SetBlock(position[0], position[1], position[2], drainedBlock)
			continue
		}
		w.SetBlock(position[0], position[1], position[2], Air)
	}
	w.SetBlock(x, y, z, Block{Namespace: "minecraft", Name: "wet_sponge"})
	return true
}

// spongeAbsorbable reports whether a block is water the sponge should remove:
// a full water block at any level, or a waterlogged block whose water is drained.
func spongeAbsorbable(block Block) bool {
	if block.ResourceLocation() == "minecraft:water" {
		return true
	}
	return block.Properties["waterlogged"] == "true"
}
