package world

// CarvePumpkin returns the carved block state for a player-facing interaction.
func CarvePumpkin(block Block, facing string) (Block, bool) {
	if block.ResourceLocation() != "minecraft:pumpkin" {
		return Block{}, false
	}
	switch facing {
	case "north", "south", "east", "west":
		return Block{Namespace: "minecraft", Name: "carved_pumpkin", Properties: map[string]string{"facing": facing}}, true
	default:
		return Block{}, false
	}
}
