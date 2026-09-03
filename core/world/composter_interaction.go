package world

import "strconv"

// AddToComposter consumes one valid item at levels zero through six. The
// returned schedule flag marks a new level-seven composter for maturation.
func AddToComposter(block Block, itemID string, x, y, z int, worldAge int64) (Block, bool, bool) {
	if block.ResourceLocation() != "minecraft:composter" || !IsCompostable(itemID) {
		return Block{}, false, false
	}
	level, err := strconv.Atoi(block.Properties["level"])
	if err != nil || level < 0 || level >= 7 {
		return Block{}, false, false
	}
	updated := copyInteractionBlock(block)
	if level == 0 || ComposterAccepts(x, y, z, worldAge, itemID) {
		level++
		updated.Properties["level"] = strconv.Itoa(level)
	}
	return updated, true, level == 7
}

// EmptyComposter collects bone meal from a ready level-eight composter.
func EmptyComposter(block Block) (Block, bool) {
	if block.ResourceLocation() != "minecraft:composter" || block.Properties["level"] != "8" {
		return Block{}, false
	}
	updated := copyInteractionBlock(block)
	updated.Properties["level"] = "0"
	return updated, true
}

func copyInteractionBlock(block Block) Block {
	properties := make(map[string]string, len(block.Properties)+1)
	for key, value := range block.Properties {
		properties[key] = value
	}
	block.Properties = properties
	return block
}
