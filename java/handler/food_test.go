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

func TestJavaGoldenAppleStoresAuthoritativeEffects(t *testing.T) {
	p := player.New([16]byte{75}, "golden-eater", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:golden_apple", Count: 1}
	started := time.Now().Add(-player.FoodUseDuration("minecraft:golden_apple"))
	if !startJavaFoodUse(p, player.HotbarStart, started) || !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("golden apple use did not complete")
	}
	regeneration, regenOK := p.StatusEffect("regeneration")
	absorption, absorptionOK := p.StatusEffect("absorption")
	if !regenOK || !absorptionOK || regeneration.Amplifier != 1 || absorption.Duration != 2400 || p.AbsorptionSnapshot() != 4 {
		t.Fatalf("stored effects = regeneration %#v absorption %#v hearts %.1f", regeneration, absorption, p.AbsorptionSnapshot())
	}
}

func TestJavaHoneyBottleCuresPoison(t *testing.T) {
	p := player.New([16]byte{76}, "honey-drinker", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:honey_bottle", Count: 1}
	p.AddStatusEffect(player.StatusEffect{ID: "poison", Duration: 100})
	started := time.Now().Add(-player.FoodUseDuration("minecraft:honey_bottle"))
	if !startJavaFoodUse(p, player.HotbarStart, started) || !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("honey bottle use did not complete")
	}
	if _, poisoned := p.StatusEffect("poison"); poisoned {
		t.Fatal("poison remained after drinking honey")
	}
	if stack := p.Inventory[player.HotbarStart]; stack.ItemID != "minecraft:glass_bottle" || stack.Count != 1 {
		t.Fatalf("honey remainder = %+v", stack)
	}
}

func TestJavaMilkBucketClearsEffects(t *testing.T) {
	p := player.New([16]byte{77}, "milk-drinker", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:milk_bucket", Count: 1}
	p.AddStatusEffect(player.StatusEffect{ID: "poison", Duration: 100})
	p.AddStatusEffect(player.StatusEffect{ID: "speed", Duration: 100})
	started := time.Now().Add(-player.FoodUseDuration("minecraft:milk_bucket"))
	if !startJavaFoodUse(p, player.HotbarStart, started) || !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("milk bucket use did not complete")
	}
	if effects := p.StatusEffectsSnapshot(); len(effects) != 0 {
		t.Fatalf("effects remain after milk: %+v", effects)
	}
	if stack := p.Inventory[player.HotbarStart]; stack.ItemID != "minecraft:bucket" || stack.Count != 1 {
		t.Fatalf("milk remainder = %+v", stack)
	}
}
