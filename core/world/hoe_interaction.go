package world

import "GoCraft/core/player"

// UseHoe returns the soil produced by a hoe and any item released by the
// transformation. canMakeFarmland carries the edition-neutral face/support
// check required by grass, dirt, and dirt paths.
func UseHoe(block Block, canMakeFarmland bool) (Block, player.ItemStack, bool) {
	switch block.ResourceLocation() {
	case "minecraft:grass_block", "minecraft:dirt", "minecraft:dirt_path":
		if !canMakeFarmland {
			return Block{}, player.ItemStack{}, false
		}
		return Block{Namespace: "minecraft", Name: "farmland", Properties: map[string]string{"moisture": "0"}}, player.ItemStack{}, true
	case "minecraft:coarse_dirt":
		return Block{Namespace: "minecraft", Name: "dirt"}, player.ItemStack{}, true
	case "minecraft:rooted_dirt":
		return Block{Namespace: "minecraft", Name: "dirt"}, player.ItemStack{ItemID: "minecraft:hanging_roots", Count: 1}, true
	default:
		return Block{}, player.ItemStack{}, false
	}
}
