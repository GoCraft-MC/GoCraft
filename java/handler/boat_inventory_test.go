package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
)

func newBoatInventoryTest(t *testing.T) (*player.Player, *corentity.Entity, *coreworld.World) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	boat := corentity.New(42, [16]byte{}, corentity.TypeOakChestBoat, 1, 64, 0)
	w.Entities.Add(boat)
	p := player.New([16]byte{1}, "sailor", player.ClientEditionJava)
	p.EntityID, p.Position = 1, boat.Position
	return p, boat, w
}

func TestChestBoatOpensFromSneakingAndMountedInventory(t *testing.T) {
	for _, mounted := range []bool{false, true} {
		p, boat, w := newBoatInventoryTest(t)
		var err error
		if mounted {
			boat.AddPassenger(p.EntityID)
			p.VehicleEntityID = boat.EntityID
			pkt := protocol.NewBuilder(packetIDPlayerCommand).VarInt(p.EntityID).VarInt(7).VarInt(0).Build()
			err = HandlePlayerCommandPacket(pkt, p, w, nil, nil)
		} else {
			pkt := protocol.NewBuilder(packetIDInteract).VarInt(boat.EntityID).VarInt(0).VarInt(0).Bool(true).Build()
			err = handleInteractPacket(pkt, p, w, nil, nil)
		}
		if err != nil {
			t.Fatal(err)
		}
		if p.OpenContainerKind != boatContainerKind || len(p.ContainerSlots) != 27 || boat.HasPassenger(p.EntityID) != mounted {
			t.Fatalf("mounted=%v: container=%s slots=%d passengers=%v", mounted, p.OpenContainerKind, len(p.ContainerSlots), boat.PassengerIDs())
		}
	}
}

func TestBoatInteractionAtDoesNotOpenChestAfterBoarding(t *testing.T) {
	p, boat, w := newBoatInventoryTest(t)
	at := protocol.NewBuilder(packetIDInteract).VarInt(boat.EntityID).VarInt(2).Float(0).Float(0).Float(0).VarInt(0).Bool(false).Build()
	use := protocol.NewBuilder(packetIDInteract).VarInt(boat.EntityID).VarInt(0).VarInt(0).Bool(false).Build()
	for _, pkt := range []*protocol.Packet{at, use} {
		if err := handleInteractPacket(pkt, p, w, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if p.VehicleEntityID != boat.EntityID || p.OpenContainerKind != "" {
		t.Fatal("boarding also opened storage")
	}
}

func TestChestBoatKeepsItemsAcrossCloseMovementAndTwoViewers(t *testing.T) {
	p, boat, w := newBoatInventoryTest(t)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 3}
	if err := openBoatInventory(p, nil, boat); err != nil {
		t.Fatal(err)
	}
	handleChestClick(p, w, 54, 0, 1)
	if !p.Inventory[player.HotbarStart].IsEmpty() {
		t.Fatal("shift-click did not move the inventory stack")
	}
	closePacket := protocol.NewBuilder(packetIDContainerClose).VarInt(chestContainerID).Build()
	if err := handleContainerClose(closePacket, p, nil, w); err != nil {
		t.Fatal(err)
	}
	boat.Position.X += 10
	p.Position = boat.Position
	if err := openBoatInventory(p, nil, boat); err != nil {
		t.Fatal(err)
	}
	if p.ContainerSlots[0].Count != 3 {
		t.Fatal("boat lost its contents after moving")
	}
	other := player.New([16]byte{2}, "viewer", player.ClientEditionJava)
	other.Position = boat.Position
	if err := openBoatInventory(other, nil, boat); err != nil {
		t.Fatal(err)
	}
	handleChestClick(p, w, 0, 0, 0)
	handleChestClick(other, w, 0, 0, 0)
	if p.CarriedItem.Count != 3 || !other.CarriedItem.IsEmpty() || !boat.Storage.Snapshot()[0].IsEmpty() {
		t.Fatal("two viewers duplicated or lost the chest stack")
	}
}
