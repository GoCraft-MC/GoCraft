package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/intent"
	"GoCraft/core/player"
)

func TestChestAttachesToTamedDonkey(t *testing.T) {
	s, p := newAnimalTestServer(t)
	donkey := corentity.New(s.game.NextEntityID(), [16]byte{7}, corentity.TypeDonkey, 1, 64, 0)
	donkey.Tamed = true
	s.world.Entities.Add(donkey)
	putHeld(p, "minecraft:chest", 1)

	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: donkey.EntityID, HotbarSlot: 0})

	if !donkey.HasChest {
		t.Fatal("chest did not attach to the tamed donkey")
	}
	if donkey.Storage == nil || len(donkey.Storage.Snapshot()) != 15 {
		t.Fatalf("donkey storage = %v, want 15 slots", donkey.Storage)
	}
	if !p.Inventory[player.HotbarStart].IsEmpty() {
		t.Fatal("chest item was not consumed")
	}
}

func TestChestDoesNotAttachToUntamedDonkey(t *testing.T) {
	s, p := newAnimalTestServer(t)
	donkey := corentity.New(s.game.NextEntityID(), [16]byte{7}, corentity.TypeDonkey, 1, 64, 0)
	donkey.Tamed = false
	s.world.Entities.Add(donkey)
	putHeld(p, "minecraft:chest", 1)

	s.applyEntityInteract(intent.EntityInteractIntent{PlayerUUID: p.UUID, TargetID: donkey.EntityID, HotbarSlot: 0})

	if donkey.HasChest {
		t.Fatal("chest attached to an untamed donkey")
	}
}

func TestChestedDonkeyDropsChestAndContentsOnDeath(t *testing.T) {
	s, _ := newAnimalTestServer(t)
	donkey := corentity.New(s.game.NextEntityID(), [16]byte{7}, corentity.TypeDonkey, 1, 64, 0)
	donkey.HasChest = true
	donkey.Storage = player.NewStorageInventory(15)
	donkey.Storage.Update(func(slots []player.ItemStack) []player.ItemStack {
		slots[0] = player.ItemStack{ItemID: "minecraft:diamond", Count: 3}
		return slots
	})
	s.world.Entities.Add(donkey)

	drops := s.spawnMobDrops(donkey)
	chest, diamond := false, false
	for _, d := range drops {
		switch d.DroppedItem().ItemID {
		case "minecraft:chest":
			chest = true
		case "minecraft:diamond":
			diamond = true
		}
	}
	if !chest {
		t.Fatal("chested donkey did not drop a chest")
	}
	if !diamond {
		t.Fatal("chested donkey did not drop its stored contents")
	}
}
