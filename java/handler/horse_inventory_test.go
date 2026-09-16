package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/protocol"
)

func newDonkeyChestTest(t *testing.T, chested bool) (*player.Player, *corentity.Entity, *coreworld.World) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	donkey := corentity.New(42, [16]byte{}, corentity.TypeDonkey, 1, 64, 0)
	donkey.Tamed = true
	if chested {
		donkey.HasChest = true
		donkey.Storage = player.NewStorageInventory(27)
	}
	w.Entities.Add(donkey)
	p := player.New([16]byte{1}, "rider", player.ClientEditionJava)
	p.EntityID, p.Position = 1, donkey.Position
	return p, donkey, w
}

func TestChestedDonkeyOpensStorageOnSneak(t *testing.T) {
	p, donkey, w := newDonkeyChestTest(t, true)
	// interact: type=0 (INTERACT), hand=0, sneaking=true
	pkt := protocol.NewBuilder(packetIDInteract).VarInt(donkey.EntityID).VarInt(0).VarInt(0).Bool(true).Build()
	if err := handleInteractPacket(pkt, p, w, nil, nil); err != nil {
		t.Fatal(err)
	}
	if p.OpenContainerKind != boatContainerKind || len(p.ContainerSlots) != 27 {
		t.Fatalf("donkey chest did not open: container=%q slots=%d", p.OpenContainerKind, len(p.ContainerSlots))
	}
}

func TestUnchestedDonkeyDoesNotOpenStorage(t *testing.T) {
	p, donkey, w := newDonkeyChestTest(t, false)
	pkt := protocol.NewBuilder(packetIDInteract).VarInt(donkey.EntityID).VarInt(0).VarInt(0).Bool(true).Build()
	if err := handleInteractPacket(pkt, p, w, nil, nil); err != nil {
		t.Fatal(err)
	}
	if p.OpenContainerKind != "" {
		t.Fatalf("chestless donkey opened a container: %q", p.OpenContainerKind)
	}
}

func TestDonkeyChestShiftClickTransfersItemFromHotbar(t *testing.T) {
	p, donkey, w := newDonkeyChestTest(t, true)
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 3}
	// Open the storage (sneak-right-click).
	open := protocol.NewBuilder(packetIDInteract).VarInt(donkey.EntityID).VarInt(0).VarInt(0).Bool(true).Build()
	if err := handleInteractPacket(open, p, w, nil, nil); err != nil {
		t.Fatal(err)
	}
	// Shift-click the first hotbar slot (container-view index 54 in a 27+36 menu).
	handleChestClick(p, w, 54, 0, 1)

	moved := false
	for _, stack := range donkey.Storage.Snapshot() {
		if stack.ItemID == "minecraft:diamond" && stack.Count == 3 {
			moved = true
		}
	}
	if !moved {
		t.Fatal("shift-click did not move the diamonds into the donkey's storage")
	}
	if !p.Inventory[player.HotbarStart].IsEmpty() {
		t.Fatal("diamonds remained in the hotbar after shift-click")
	}
}

func TestRiddenChestedDonkeyOpensStorageWithInventoryKey(t *testing.T) {
	p, donkey, w := newDonkeyChestTest(t, true)
	donkey.AddPassenger(p.EntityID)
	p.VehicleEntityID = donkey.EntityID
	// PlayerCommand action 7 = OPEN_INVENTORY while riding.
	pkt := protocol.NewBuilder(packetIDPlayerCommand).VarInt(p.EntityID).VarInt(7).VarInt(0).Build()
	if err := HandlePlayerCommandPacket(pkt, p, w, nil, nil); err != nil {
		t.Fatal(err)
	}
	if p.OpenContainerKind != boatContainerKind || len(p.ContainerSlots) != 27 {
		t.Fatalf("ridden donkey chest did not open: container=%q slots=%d", p.OpenContainerKind, len(p.ContainerSlots))
	}
}
