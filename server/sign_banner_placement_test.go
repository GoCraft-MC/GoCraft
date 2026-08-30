package server

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestBedrockSignsAndBannersCreateCanonicalBlockEntities(t *testing.T) {
	for _, test := range []struct {
		item, wantBlock, wantEntity string
	}{
		{"minecraft:oak_sign", "minecraft:oak_sign", "minecraft:sign"},
		{"minecraft:red_banner", "minecraft:red_banner", "minecraft:banner"},
	} {
		t.Run(test.item, func(t *testing.T) {
			s, p := newBedrockActionTestServer(t)
			s.world.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: test.item, Count: 1}
			s.applyBedrockBlockInteract(intent.BlockInteractIntent{
				PlayerUUID: p.UUID, Action: intent.BlockActionUse,
				Position: spatial.BlockPos{X: 0, Y: 64, Z: 0}, Face: 1, HotbarSlot: 0,
				ClickX: 0.5, ClickY: 1, ClickZ: 0.5,
			})
			if got := s.world.GetBlock(0, 65, 0).ResourceLocation(); got != test.wantBlock {
				t.Fatalf("placed block = %q, want %q", got, test.wantBlock)
			}
			entities := s.world.LoadedBlockEntities()
			if len(entities) != 1 || entities[0].Type != test.wantEntity {
				t.Fatalf("block entities = %+v, want one %s", entities, test.wantEntity)
			}
		})
	}
}
