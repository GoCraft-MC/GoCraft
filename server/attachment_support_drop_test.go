package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

func TestBedrockSupportBreakDropsAttachmentItem(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	g := game.New()
	p := player.New([16]byte{1}, "builder", player.ClientEditionBedrock)
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{world: w, game: g}

	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(1, 64, 0, coreworld.Block{
		Namespace:  "minecraft",
		Name:       "oak_wall_sign",
		Properties: map[string]string{"facing": "east"},
	})
	w.SetBlock(0, 64, 0, coreworld.Air)
	s.breakBedrockUnsupportedAbove(p, 0, 64, 0)

	for _, entity := range w.Entities.Snapshot() {
		if entity.ItemID == "minecraft:oak_sign" && entity.ItemCount == 1 {
			return
		}
	}
	t.Fatal("support-broken wall sign did not spawn an oak sign item")
}
