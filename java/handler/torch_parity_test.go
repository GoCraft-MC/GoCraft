package handler

import (
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
)

func TestJavaTorchCannotAttachToCeiling(t *testing.T) {
	p := player.New([16]byte{}, "builder", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:torch", Count: 1}
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(0, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	pkt := protocol.NewBuilder(packetIDUseItemOn).
		VarInt(0).
		Long(packBlockPos(0, 65, 0)).
		VarInt(0).
		Float(0.5).
		Float(0.0).
		Float(0.5).
		Bool(false).
		Bool(false).
		VarInt(901).
		Build()
	if err := handleUseItemOn(pkt, p, w, mgr, nil, nil); err != nil {
		t.Fatalf("handleUseItemOn: %v", err)
	}
	if got := w.GetBlock(0, 64, 0); !got.IsAir() {
		t.Fatalf("ceiling placement produced %q, want air", got.ResourceLocation())
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("rejected ceiling placement consumed torch: count=%d, want 1", got)
	}
}

func TestJavaWallTorchBreakDropsStandingTorch(t *testing.T) {
	p := player.New([16]byte{}, "survivor", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	mgr := session.NewManager()
	w.SetBlock(1, 64, 0, coreworld.Block{
		Namespace:  "minecraft",
		Name:       "wall_torch",
		Properties: map[string]string{"facing": "east"},
	})

	pkt := protocol.NewBuilder(packetIDPlayerAction).
		VarInt(actionStatusFinishDigging).
		Long(packBlockPos(1, 64, 0)).
		Byte(5).
		VarInt(902).
		Build()
	if err := handlePlayerAction(pkt, p, w, mgr); err != nil {
		t.Fatalf("handlePlayerAction: %v", err)
	}
	if got := w.GetBlock(1, 64, 0); !got.IsAir() {
		t.Fatalf("broken wall torch = %q, want air", got.ResourceLocation())
	}
	if !javaWorldHasDroppedItem(w, "minecraft:torch", 1) {
		t.Fatal("breaking wall torch did not drop one minecraft:torch")
	}
}
