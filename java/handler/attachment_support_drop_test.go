package handler

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestJavaSupportBreakDropsAttachmentItem(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(1, 64, 0, coreworld.Block{
		Namespace:  "minecraft",
		Name:       "oak_wall_sign",
		Properties: map[string]string{"facing": "east"},
	})
	w.SetBlock(0, 64, 0, coreworld.Air)

	nextID := int32(1000)
	breakUnsupportedBlocksAboveWithDrops(0, 64, 0, w, nil, func() int32 {
		nextID++
		return nextID
	}, 0)

	for _, entity := range w.Entities.Snapshot() {
		if entity.ItemID == "minecraft:oak_sign" && entity.ItemCount == 1 {
			return
		}
	}
	t.Fatal("support-broken wall sign did not spawn an oak sign item")
}
