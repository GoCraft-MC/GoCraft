package bedrock

import (
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestNormalInventoryTransactionDropsOneHotbarItem(t *testing.T) {
	p := player.New([16]byte{71}, "dropper", player.ClientEditionBedrock)
	p.Inventory[player.HotbarStart+2] = player.ItemStack{ItemID: "minecraft:stone", Count: 4}
	oldStack, ok := bedrockJavaOutput(p.Inventory[player.HotbarStart+2])
	if !ok {
		t.Fatal("stone did not encode as a Bedrock item")
	}
	remaining := oldStack
	remaining.Count = 3
	dropped := oldStack
	dropped.Count = 1

	pk := &packet.InventoryTransaction{
		TransactionData: &protocol.NormalTransactionData{},
		Actions: []protocol.InventoryAction{
			{
				SourceType:    protocol.InventoryActionSourceWorld,
				SourceFlags:   protocol.Option(uint32(0)),
				InventorySlot: 0,
				NewItem:       protocol.ItemInstance{Stack: dropped},
			},
			{
				SourceType:    protocol.InventoryActionSourceContainer,
				WindowID:      protocol.Option(int8(protocol.WindowIDInventory)),
				InventorySlot: 2,
				OldItem:       protocol.ItemInstance{Stack: oldStack},
				NewItem:       protocol.ItemInstance{Stack: remaining},
			},
		},
	}
	actions, slots, valid := canonicalNormalDropActions(p, pk)
	if !valid || len(actions) != 1 || len(slots) != 1 {
		t.Fatalf("drop translation = valid %v, actions %+v, slots %v", valid, actions, slots)
	}
	if got := actions[0]; got.Kind != intent.InventoryActionDrop || got.Source != player.HotbarStart+2 || got.Count != 1 {
		t.Fatalf("translated drop = %+v", got)
	}
}

func TestNormalInventoryTransactionDropsCompleteStack(t *testing.T) {
	p := player.New([16]byte{72}, "stack-dropper", player.ClientEditionBedrock)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 16}
	oldStack, ok := bedrockJavaOutput(p.Inventory[player.HotbarStart])
	if !ok {
		t.Fatal("oak log did not encode as a Bedrock item")
	}

	pk := &packet.InventoryTransaction{Actions: []protocol.InventoryAction{
		{
			SourceType:    protocol.InventoryActionSourceContainer,
			WindowID:      protocol.Option(int8(protocol.WindowIDInventory)),
			InventorySlot: 0,
			OldItem:       protocol.ItemInstance{Stack: oldStack},
		},
		{
			SourceType:    protocol.InventoryActionSourceWorld,
			SourceFlags:   protocol.Option(uint32(0)),
			InventorySlot: 0,
			NewItem:       protocol.ItemInstance{Stack: oldStack},
		},
	}}
	actions, _, valid := canonicalNormalDropActions(p, pk)
	if !valid || len(actions) != 1 || actions[0].Count != 16 {
		t.Fatalf("full-stack translation = valid %v, actions %+v", valid, actions)
	}
}

func TestLegacyNormalInventoryTransactionMatchesPumpkinFullStackDrop(t *testing.T) {
	p := player.New([16]byte{73}, "legacy-dropper", player.ClientEditionBedrock)
	p.Inventory[player.HotbarStart+4] = player.ItemStack{ItemID: "minecraft:apple", Count: 7}
	pk := &packet.InventoryTransaction{
		LegacyRequestID: -2,
		LegacySetItemSlots: []protocol.LegacySetItemSlot{{
			ContainerID: protocol.ContainerHotBar,
			Slots:       []byte{4},
		}},
		TransactionData: &protocol.NormalTransactionData{},
	}
	actions, slots, valid := canonicalNormalDropActions(p, pk)
	if !valid || len(actions) != 1 || actions[0].Source != player.HotbarStart+4 || actions[0].Count != 7 || len(slots) != 1 {
		t.Fatalf("legacy drop translation = valid %v, actions %+v, slots %v", valid, actions, slots)
	}
}

func TestNormalInventoryTransactionRejectsUnbalancedDrop(t *testing.T) {
	p := player.New([16]byte{74}, "invalid-dropper", player.ClientEditionBedrock)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 3}
	oldStack, ok := bedrockJavaOutput(p.Inventory[player.HotbarStart])
	if !ok {
		t.Fatal("diamond did not encode as a Bedrock item")
	}
	dropped := oldStack
	dropped.Count = 2
	remaining := oldStack
	remaining.Count = 2 // Invalid: 3 - 2 must leave 1.
	pk := &packet.InventoryTransaction{Actions: []protocol.InventoryAction{
		{
			SourceType: protocol.InventoryActionSourceContainer,
			WindowID:   protocol.Option(int8(protocol.WindowIDInventory)),
			OldItem:    oldStackInstance(oldStack), NewItem: oldStackInstance(remaining),
		},
		{
			SourceType:  protocol.InventoryActionSourceWorld,
			SourceFlags: protocol.Option(uint32(0)),
			NewItem:     oldStackInstance(dropped),
		},
	}}
	if actions, _, valid := canonicalNormalDropActions(p, pk); valid || len(actions) != 0 {
		t.Fatalf("unbalanced transaction accepted: %+v", actions)
	}
}

func oldStackInstance(stack protocol.ItemStack) protocol.ItemInstance {
	return protocol.ItemInstance{Stack: stack}
}
