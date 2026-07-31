package handler

import (
	"strings"
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func TestCreativeInventoryUsesProtocol769ItemIDs(t *testing.T) {
	tests := []struct {
		itemID int32
		name   string
	}{
		{40, "minecraft:acacia_planks"},
		{195, "minecraft:glass"},
		{314, "minecraft:crafting_table"},
		{316, "minecraft:furnace"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := player.New([16]byte{}, "builder", player.ClientEditionJava)
			pkt := protocol.NewBuilder(packetIDCreativeModeSetItem).
				Short(player.HotbarStart).
				VarInt(64).
				VarInt(tc.itemID).
				VarInt(0).
				VarInt(0).
				Build()
			if err := handleCreativeModeSetItem(pkt, p); err != nil {
				t.Fatalf("handleCreativeModeSetItem: %v", err)
			}
			got := p.Inventory[player.HotbarStart]
			if got.ItemID != tc.name || got.Count != 64 {
				t.Fatalf("hotbar item = %+v, want ItemID=%q Count=64", got, tc.name)
			}
		})
	}
}

func TestUseItemOnProtocol769LayoutPlacesExactBlock(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:acacia_planks", Count: 64}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()

	pkt := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).
		Long(packBlockPos(0, 63, 0)).
		VarInt(1).
		Float(0.5).
		Float(1.0).
		Float(0.5).
		Bool(false).
		Bool(false).
		VarInt(300).
		Build()
	if err := handleUseItemOn(pkt, p, w, mgr, nil); err != nil {
		t.Fatalf("handleUseItemOn: %v", err)
	}
	got := w.GetBlock(0, 64, 0)
	if got.ResourceLocation() != "minecraft:acacia_planks" {
		t.Fatalf("placed block = %q, want minecraft:acacia_planks", got.ResourceLocation())
	}
}

func TestUseItemOnRequiresSequenceAfterWorldBorderHit(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()

	pkt := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).
		Long(packBlockPos(0, 63, 0)).
		VarInt(1).
		Float(0.5).
		Float(1.0).
		Float(0.5).
		Bool(false).
		Bool(false).
		Build()
	err := handleUseItemOn(pkt, p, w, mgr, nil)
	if err == nil || !strings.Contains(err.Error(), "sequence") {
		t.Fatalf("handleUseItemOn error = %v, want missing sequence error after world_border_hit", err)
	}
}

func TestBreakingOneGlassBlockDoesNotBreakAnother(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	mgr := session.NewManager()
	glass := coreworld.Block{Namespace: "minecraft", Name: "glass"}
	w.SetBlock(1, 64, 0, glass)
	w.SetBlock(2, 64, 0, glass)

	pkt := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusStartDigging).
		Long(packBlockPos(1, 64, 0)).
		Byte(1).
		VarInt(301).
		Build()
	if err := handlePlayerAction(pkt, p, w, mgr); err != nil {
		t.Fatalf("handlePlayerAction: %v", err)
	}
	if got := w.GetBlock(1, 64, 0); !got.IsAir() {
		t.Fatalf("target block = %q, want air", got.ResourceLocation())
	}
	if got := w.GetBlock(2, 64, 0); !got.Equal(glass) {
		t.Fatalf("neighbor block = %q, want glass", got.ResourceLocation())
	}
}
