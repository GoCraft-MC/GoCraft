package handler

import (
	"testing"
	"time"

	"GoCraft/core/player"
)

func TestJavaFoodUseCompletesAndConsumesHeldStack(t *testing.T) {
	p := player.New([16]byte{72}, "java-eater", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Food = 10
	p.Saturation = 0
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bread", Count: 2}
	started := time.Now().Add(-player.FoodUseDuration("minecraft:bread"))
	if !startJavaFoodUse(p, player.HotbarStart, started) {
		t.Fatal("Java food use did not start")
	}
	if !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("Java food use did not complete")
	}
	food, saturation, _ := p.HungerSnapshot()
	if food != 15 || saturation != 6 {
		t.Fatalf("hunger after bread = %d/%.1f, want 15/6", food, saturation)
	}
	if stack := p.Inventory[player.HotbarStart]; stack.ItemID != "minecraft:bread" || stack.Count != 1 {
		t.Fatalf("held stack after eating = %+v", stack)
	}
	if p.UsingItemID != "" || p.UsingItemSlot != -1 || !p.UsingItemSince.IsZero() {
		t.Fatalf("food use state was not cleared: %q/%d/%v", p.UsingItemID, p.UsingItemSlot, p.UsingItemSince)
	}
}

func TestJavaFoodReleaseBeforeDurationCancelsUse(t *testing.T) {
	p := player.New([16]byte{73}, "java-eater", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Food = 10
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:cooked_beef", Count: 1}
	if !startJavaFoodUse(p, player.HotbarStart, time.Now()) {
		t.Fatal("Java food use did not start")
	}
	releaseRangedItem(p, nil, nil, nil, nil)
	if p.UsingItemID != "" || p.Inventory[player.HotbarStart].Count != 1 || p.Food != 10 {
		t.Fatalf("early release consumed food: item=%q stack=%+v food=%d", p.UsingItemID, p.Inventory[player.HotbarStart], p.Food)
	}
}

func TestJavaStewLeavesBowl(t *testing.T) {
	p := player.New([16]byte{74}, "java-eater", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Food = 10
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:mushroom_stew", Count: 1}
	started := time.Now().Add(-player.FoodUseDuration("minecraft:mushroom_stew"))
	if !startJavaFoodUse(p, player.HotbarStart, started) || !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("stew use did not complete")
	}
	if stack := p.Inventory[player.HotbarStart]; stack.ItemID != "minecraft:bowl" || stack.Count != 1 {
		t.Fatalf("stew remainder = %+v, want bowl", stack)
	}
}
