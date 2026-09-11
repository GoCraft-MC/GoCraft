package world

import "strconv"

// Twisting and weeping vines share vanilla's growing-plant-head mechanic: a tip
// block with age 0..25 grows into air, leaving a "_plant" body behind. Twisting
// vines climb upward; weeping vines hang downward.
const netherVineGrowthSalt = uint64(0x5deece66d1234567)

func isNetherVineHead(name string) bool {
	return name == "minecraft:twisting_vines" || name == "minecraft:weeping_vines"
}

// netherVineDirection returns the vertical step a vine tip grows towards.
func netherVineDirection(name string) int {
	if name == "minecraft:weeping_vines" {
		return -1
	}
	return 1
}

func netherVineBody(name string) string {
	return name[len("minecraft:"):] + "_plant"
}

// tickNetherVineAt advances a twisting or weeping vine tip into adjacent air,
// carrying the incremented age to the new tip and converting the old tip to a
// body segment. Growth is gated so a vine lengthens gradually.
func (w *World) tickNetherVineAt(x, y, z int, vine Block, tick int64) []BlockChange {
	name := vine.ResourceLocation()
	age := kelpAge(vine)
	if age >= kelpMaxAge {
		return nil
	}
	dy := netherVineDirection(name)
	target, loaded := w.blockIfLoaded(x, y+dy, z)
	if !loaded || !target.IsAir() {
		return nil
	}
	seed := uint64(tick / 20)
	if cropRandom(seed, x, y, z, netherVineGrowthSalt, 10) != 0 {
		return nil
	}
	newTip := Block{Namespace: "minecraft", Name: name[len("minecraft:"):], Properties: map[string]string{"age": strconv.Itoa(age + 1)}}
	body := Block{Namespace: "minecraft", Name: netherVineBody(name)}
	w.SetBlock(x, y+dy, z, newTip)
	w.SetBlock(x, y, z, body)
	return []BlockChange{
		{X: x, Y: y + dy, Z: z, Block: newTip},
		{X: x, Y: y, Z: z, Block: body},
	}
}
