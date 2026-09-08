package handler

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreplugin "GoCraft/core/plugin"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
)

func TestEntityInteractRunsOnceAndMarksForwardedIntent(t *testing.T) {
	p, boat, w := newBoatInventoryTest(t)
	intents := intent.NewBus(1, 4)
	cancel, calls := true, 0
	bus := testEventBus(t, coreplugin.EventPlayerInteract, func(e *abi.Event) abi.Verdict { calls++; return abi.Verdict{Cancelled: cancel} })
	point := protocol.NewBuilder(packetIDInteract).VarInt(boat.EntityID).VarInt(2).Float(0).Float(0).Float(0).VarInt(0).Bool(false).Build()
	use := protocol.NewBuilder(packetIDInteract).VarInt(boat.EntityID).VarInt(0).VarInt(0).Bool(false).Build()
	for _, pkt := range []*protocol.Packet{point, use} {
		if err := handleInteractPacketWithEvents(pkt, p, w, nil, nil, bus, intents); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 || len(intents.Drain().Gameplay) != 0 {
		t.Fatal("cancel ignored or duplicate event")
	}
	cancel = false
	if err := handleInteractPacketWithEvents(use, p, w, nil, nil, bus, intents); err != nil {
		t.Fatal(err)
	}
	queued := intents.Drain().Gameplay
	if len(queued) != 1 || !queued[0].(intent.EntityInteractIntent).EventChecked || calls != 2 {
		t.Fatal("lost dispatch marker")
	}
}

func TestCancelledInventoryClickAndItemUseKeepStacks(t *testing.T) {
	p, _, w := newBoatInventoryTest(t)
	p.CarriedItem = player.ItemStack{ItemID: "minecraft:diamond", Count: 3}
	p.QuickCraftSlots = []int{9}
	conn := network.NewClientConn(&boatPacketSink{})
	bus := testEventBus(t, coreplugin.EventInventoryClick, func(e *abi.Event) abi.Verdict { return abi.Verdict{Cancelled: true} })
	click := protocol.NewBuilder(packetIDContainerClick).VarInt(0).VarInt(0).Short(9).Byte(0).VarInt(0).VarInt(0).VarInt(0).Build()
	if err := handleContainerClick(click, p, conn, w, nil, bus); err != nil {
		t.Fatal(err)
	}
	if !p.Inventory[9].IsEmpty() || p.CarriedItem.Count != 3 || len(p.QuickCraftSlots) != 0 {
		t.Fatal("cancelled click changed stacks")
	}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:snowball", Count: 3}
	bus = testEventBus(t, coreplugin.EventItemUse, func(e *abi.Event) abi.Verdict { return abi.Verdict{Cancelled: true} })
	use := protocol.NewBuilder(packetIDUseItem).VarInt(0).VarInt(1).Float(0).Float(0).Build()
	if err := handleUseItem(use, p, conn, w, nil, nil, bus); err != nil {
		t.Fatal(err)
	}
	if p.HeldItem().Count != 3 {
		t.Fatal("cancelled item was consumed")
	}
}
