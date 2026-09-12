package world

import "strconv"

// ChargeRespawnAnchor adds one glowstone charge up to the vanilla maximum.
func ChargeRespawnAnchor(block Block, itemID string) (Block, bool) {
	if block.ResourceLocation() != "minecraft:respawn_anchor" || itemID != "minecraft:glowstone" {
		return Block{}, false
	}
	charges, err := strconv.Atoi(block.Properties["charges"])
	if err != nil {
		charges = 0
	}
	if charges < 0 || charges >= 4 {
		return Block{}, false
	}
	updated := copyInteractionBlock(block)
	updated.Properties["charges"] = strconv.Itoa(charges + 1)
	return updated, true
}
