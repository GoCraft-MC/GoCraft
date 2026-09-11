package world

import "strings"

// WaxCopper returns the vanilla waxed variant while preserving block state.
func WaxCopper(block Block) (Block, bool) {
	if block.Namespace != "minecraft" || strings.HasPrefix(block.Name, "waxed_") {
		return Block{}, false
	}
	base := block.Name
	for _, stage := range []string{"exposed_", "weathered_", "oxidized_"} {
		base = strings.TrimPrefix(base, stage)
	}
	switch base {
	case "copper_block", "cut_copper", "cut_copper_stairs", "cut_copper_slab",
		"chiseled_copper", "copper_grate", "copper_bulb", "copper_door", "copper_trapdoor":
		properties := make(map[string]string, len(block.Properties))
		for key, value := range block.Properties {
			properties[key] = value
		}
		block.Name = "waxed_" + block.Name
		block.Properties = properties
		return block, true
	default:
		return Block{}, false
	}
}
