package world

import "strings"

// coralDeathDelay matches vanilla: coral cut off from water dies after 60 ticks.
const coralDeathDelay = 60

// IsLiveCoral reports whether a block is a living coral plant, fan, or block
// that dies when it loses contact with water.
func IsLiveCoral(name string) bool {
	if !strings.Contains(name, "coral") || strings.HasPrefix(name, "minecraft:dead_") {
		return false
	}
	return strings.HasSuffix(name, "_coral") || strings.HasSuffix(name, "_coral_block") ||
		strings.HasSuffix(name, "_coral_fan") || strings.HasSuffix(name, "_coral_wall_fan")
}

// coralHasWater reports whether a coral position is still touching water, either
// by being waterlogged or by any of its six neighbours being a water block.
func (w *World) coralHasWater(x, y, z int) bool {
	if w.GetBlock(x, y, z).Properties["waterlogged"] == "true" {
		return true
	}
	for _, pos := range neighbors6(x, y, z) {
		if w.GetBlock(pos[0], pos[1], pos[2]).ResourceLocation() == "minecraft:water" {
			return true
		}
	}
	return false
}

// ApplyCoralDeath converts a live coral to its dead variant when it is no longer
// touching water. Coral still in contact with water survives unchanged.
func (w *World) ApplyCoralDeath(x, y, z int) (BlockChange, bool) {
	block := w.GetBlock(x, y, z)
	if !IsLiveCoral(block.ResourceLocation()) || w.coralHasWater(x, y, z) {
		return BlockChange{}, false
	}
	dead := copyWorldBlock(block)
	dead.Name = "dead_" + block.Name
	w.SetBlock(x, y, z, dead)
	return BlockChange{X: x, Y: y, Z: z, Block: dead}, true
}
