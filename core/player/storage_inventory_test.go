package player

import (
	"sync"
	"testing"
)

func TestStorageTransactionsUseLatestContents(t *testing.T) {
	storage := NewStorageInventory(27)
	storage.Update(func(slots []ItemStack) []ItemStack {
		slots[0] = ItemStack{ItemID: "minecraft:diamond", Count: 1}
		return slots
	})
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			storage.Update(func(slots []ItemStack) []ItemStack {
				slots[0].Count++
				return slots
			})
		})
	}
	workers.Wait()
	view := storage.Snapshot()
	if view[0].Count != 21 {
		t.Fatalf("count = %d, want 21", view[0].Count)
	}
	view[0].Count = 100
	if storage.Snapshot()[0].Count != 21 {
		t.Fatal("screen mutated stored contents")
	}
	drops := storage.Drain()
	if drops[0].Count != 21 || len(storage.Drain()) != 0 {
		t.Fatal("drain lost or duplicated items")
	}
	if storage.Update(func(slots []ItemStack) []ItemStack { t.Fatal("removed inventory was modified"); return slots }) {
		t.Fatal("removed inventory accepted a transaction")
	}
}
