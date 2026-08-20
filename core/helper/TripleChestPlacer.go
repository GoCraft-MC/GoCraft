package helper

import (
	coreworld "GoCraft/core/world"
)

type TripleChestPlacer struct {
	World *coreworld.World
}

func (t *TripleChestPlacer) Place(x, y, z int) {
	chest := coreworld.Block{
		Namespace: "minecraft",
		Name:      "Chest",
		Properties: map[string]string{
			"facing":      "north",
			"type":        "single",
			"waterlogged": "false",
		},
	}
	t.World.SetBlock(x, y, z, chest)
	t.World.SetBlock(x+1, y, z, chest)
	t.World.SetBlock(x+2, y, z, chest)
}
