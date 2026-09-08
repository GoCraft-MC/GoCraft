package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
)

func TestDestroyedChestBoatDropsItsContents(t *testing.T) {
	s, _ := newAnimalTestServer(t)
	boat := corentity.New(42, [16]byte{}, corentity.TypeOakChestBoat, 0, 64, 0)
	boat.Storage.Update(func(slots []player.ItemStack) []player.ItemStack {
		slots[0] = player.ItemStack{ItemID: "minecraft:diamond", Count: 3}
		return slots
	})
	boat.Damage(boat.MaxHealth)
	drops := s.spawnMobDrops(boat)
	counts := make(map[string]int)
	for _, drop := range drops {
		counts[drop.ItemID] += drop.ItemCount
	}
	if counts["minecraft:oak_chest_boat"] != 1 || counts["minecraft:diamond"] != 3 {
		t.Fatalf("drops = %v", counts)
	}
	if len(boat.Storage.Snapshot()) != 0 {
		t.Fatal("destroyed boat retained items")
	}
}
