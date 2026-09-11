package handler

import (
	"testing"

	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
	"GoCraft/java/session"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestJavaPlacementCancellationPreservesLinkedBlocksAndItems(t *testing.T) {
	for _, item := range []string{"stone", "oak_door", "red_bed", "chest", "decorated_pot"} {
		t.Run(item, func(t *testing.T) {
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			p := player.New([16]byte{1}, "builder", player.ClientEditionJava)
			p.GameMode = player.GameModeSurvival
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:" + item, Count: 3}
			plant := coreworld.Block{Namespace: "minecraft", Name: "tall_grass", Properties: map[string]string{"half": "lower"}}
			upper := copyBlockProperties(plant)
			upper.Properties["half"] = "upper"
			w.SetBlock(0, 64, 0, plant)
			w.SetBlock(0, 65, 0, upper)
			calls := 0
			bus := testEventBus(t, coreplugin.EventBlockPlace, func(e *abi.Event) abi.Verdict {
				calls++
				if e.Fields[2].List[0].String != "minecraft:"+item {
					t.Fatal(e.Fields[2])
				}
				if !w.GetBlock(0, 65, 0).Equal(upper) {
					t.Fatal("linked plant changed before event")
				}
				return abi.Verdict{Cancelled: true}
			})
			pkt := protocol.NewBuilder(packetIDUseItemOn).VarInt(0).Long(packBlockPos(0, 63, 0)).VarInt(1).
				Float(0.5).Float(1).Float(0.5).Bool(false).Bool(false).VarInt(1).Build()
			if err := handleUseItemOnWithIntents(pkt, p, w, session.NewManager(), nil, nil, nil, bus); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || p.HeldItem().Count != 3 || !w.GetBlock(0, 64, 0).Equal(plant) || !w.GetBlock(0, 65, 0).Equal(upper) {
				t.Fatalf("cancel failed: calls=%d count=%d", calls, p.HeldItem().Count)
			}
			if !w.GetBlock(0, 64, 1).IsAir() {
				t.Fatal("bed partner was placed")
			}
		})
	}
}
