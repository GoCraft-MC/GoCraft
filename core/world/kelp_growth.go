package world

import "strconv"

// Kelp grows a column upward through water. The growing tip is minecraft:kelp
// with an age 0..25; every segment left behind becomes minecraft:kelp_plant.
// Growth stops at age 25 or when the column reaches non-source water.
const kelpMaxAge = 25

// kelpGrowthSalt paces kelp growth roughly like vanilla's 14% per random tick.
const kelpGrowthSalt = uint64(0xc0ffee1234567890)

func kelpAge(block Block) int {
	age, err := strconv.Atoi(block.Properties["age"])
	if err != nil || age < 0 {
		return 0
	}
	if age > kelpMaxAge {
		return kelpMaxAge
	}
	return age
}

// tickKelpAt advances the kelp tip. It grows into a water source directly above,
// leaving a kelp_plant body behind and carrying the incremented age to the new
// tip. Growth is gated so a column takes many ticks to reach full height.
func (w *World) tickKelpAt(x, y, z int, kelp Block, tick int64) []BlockChange {
	age := kelpAge(kelp)
	if age >= kelpMaxAge {
		return nil
	}
	above, loaded := w.blockIfLoaded(x, y+1, z)
	if !loaded || above.ResourceLocation() != "minecraft:water" || FluidLevel(above) != 0 {
		return nil
	}
	seed := uint64(tick / 20)
	if cropRandom(seed, x, y, z, kelpGrowthSalt, 7) != 0 {
		return nil
	}
	newTip := Block{Namespace: "minecraft", Name: "kelp", Properties: map[string]string{"age": strconv.Itoa(age + 1)}}
	body := Block{Namespace: "minecraft", Name: "kelp_plant"}
	w.SetBlock(x, y+1, z, newTip)
	w.SetBlock(x, y, z, body)
	return []BlockChange{
		{X: x, Y: y + 1, Z: z, Block: newTip},
		{X: x, Y: y, Z: z, Block: body},
	}
}
