package blockloot

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestWallTorchesUseStandingTorchLoot(t *testing.T) {
	tests := []struct {
		block string
		item  string
	}{
		{"wall_torch", "minecraft:torch"},
		{"soul_wall_torch", "minecraft:soul_torch"},
		{"redstone_wall_torch", "minecraft:redstone_torch"},
	}
	for _, tc := range tests {
		t.Run(tc.block, func(t *testing.T) {
			drops := Drops(Context{Block: coreworld.Block{Namespace: "minecraft", Name: tc.block}})
			if len(drops) != 1 || drops[0].ItemID != tc.item || drops[0].Count != 1 {
				t.Fatalf("drops = %+v, want one %s", drops, tc.item)
			}
		})
	}
}
