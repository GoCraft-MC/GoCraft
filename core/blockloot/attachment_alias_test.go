package blockloot

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestAttachmentRuntimeVariantsUseInventoryItemLoot(t *testing.T) {
	tests := []struct {
		block string
		item  string
	}{
		{"oak_wall_sign", "minecraft:oak_sign"},
		{"oak_wall_hanging_sign", "minecraft:oak_hanging_sign"},
		{"white_wall_banner", "minecraft:white_banner"},
		{"skeleton_wall_skull", "minecraft:skeleton_skull"},
		{"player_wall_head", "minecraft:player_head"},
	}
	for _, tc := range tests {
		t.Run(tc.block, func(t *testing.T) {
			drops := Drops(Context{Block: coreworld.Block{Namespace: "minecraft", Name: tc.block}})
			if len(drops) != 1 || drops[0].ItemID != tc.item || drops[0].Count != 1 {
				t.Fatalf("drops = %+v, want one %s", drops, tc.item)
			}
		})
	}
	db := data()
	if _, ok := db.tables["minecraft:tube_coral_wall_fan"]; !ok {
		t.Fatal("tube coral wall fan loot table alias is missing")
	}
}
