package world

import "strconv"

// Sugar cane and cactus grow by advancing a per-block age counter from 0 to 15.
// When the counter reaches 15 the plant grows one block upward (resetting the
// base to age 0), bounded to a total column height of three. This mirrors
// vanilla's random-tick counter; GoCraft advances it once per crop scan.
const tallPlantMaxHeight = 3

// isTallPlantGrowth reports whether a block grows through the tall-plant age
// counter rather than the crop age system.
func isTallPlantGrowth(name string) bool {
	return name == "minecraft:sugar_cane" || name == "minecraft:cactus"
}

// tallPlantAge reads the 0..15 age counter, treating malformed values as zero.
func tallPlantAge(block Block) int {
	age, err := strconv.Atoi(block.Properties["age"])
	if err != nil || age < 0 {
		return 0
	}
	if age > 15 {
		return 15
	}
	return age
}

func setTallPlantAge(block Block, age int) Block {
	if age < 0 {
		age = 0
	}
	if age > 15 {
		age = 15
	}
	block = copyWorldBlock(block)
	block.Properties["age"] = strconv.Itoa(age)
	return block
}

// tickTallPlantAt advances one sugar cane or cactus. Growth requires air above
// and a column shorter than three; cactus additionally refuses to grow when a
// solid block sits beside the new position, matching its survival rule.
func (w *World) tickTallPlantAt(x, y, z int, plant Block) []BlockChange {
	name := plant.ResourceLocation()
	above, loaded := w.blockIfLoaded(x, y+1, z)
	if !loaded || !above.IsAir() {
		return nil
	}
	height := 1
	for offset := 1; offset < tallPlantMaxHeight; offset++ {
		below, ok := w.blockIfLoaded(x, y-offset, z)
		if !ok || below.ResourceLocation() != name {
			break
		}
		height++
	}
	if height >= tallPlantMaxHeight {
		return nil
	}
	age := tallPlantAge(plant)
	if age < 15 {
		grown := setTallPlantAge(plant, age+1)
		w.SetBlock(x, y, z, grown)
		return []BlockChange{{X: x, Y: y, Z: z, Block: grown}}
	}
	if name == "minecraft:cactus" && !w.cactusGrowthClear(x, y+1, z) {
		// Leave the base at age 15; it retries once its surroundings clear.
		return nil
	}
	newTop := Block{Namespace: "minecraft", Name: name[len("minecraft:"):], Properties: map[string]string{"age": "0"}}
	base := setTallPlantAge(plant, 0)
	w.SetBlock(x, y+1, z, newTop)
	w.SetBlock(x, y, z, base)
	return []BlockChange{
		{X: x, Y: y + 1, Z: z, Block: newTop},
		{X: x, Y: y, Z: z, Block: base},
	}
}

// cactusGrowthClear reports whether the four horizontal neighbours of a
// position are free of solid blocks, which is required for a cactus to survive
// there. Unloaded neighbours are treated as clear so growth is never blocked by
// terrain the server has not paged in.
func (w *World) cactusGrowthClear(x, y, z int) bool {
	for _, direction := range horizontalCropDirections {
		neighbor, loaded := w.blockIfLoaded(x+direction.dx, y, z+direction.dz)
		if !loaded {
			continue
		}
		if !neighbor.IsAir() && IsEntitySupportBlock(neighbor.ResourceLocation()) {
			return false
		}
	}
	return true
}
