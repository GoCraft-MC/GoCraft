package world

import "strings"

// AddCandleToCake converts an untouched cake and a vanilla candle item into
// the matching unlit candle-cake block.
func AddCandleToCake(cake Block, itemID string) (Block, bool) {
	if cake.ResourceLocation() != "minecraft:cake" {
		return Block{}, false
	}
	name := strings.TrimPrefix(itemID, "minecraft:")
	if name == "candle" {
		return Block{Namespace: "minecraft", Name: "candle_cake", Properties: map[string]string{"lit": "false"}}, true
	}
	color := strings.TrimSuffix(name, "_candle")
	if color == name {
		return Block{}, false
	}
	switch color {
	case "white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black":
		return Block{Namespace: "minecraft", Name: color + "_candle_cake", Properties: map[string]string{"lit": "false"}}, true
	default:
		return Block{}, false
	}
}
