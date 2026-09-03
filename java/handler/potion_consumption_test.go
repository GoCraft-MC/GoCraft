package handler

import (
	"testing"
	"time"

	"GoCraft/core/player"
)

func TestJavaDrinkablePotionAppliesPayloadAndReturnsBottle(t *testing.T) {
	p := player.New([16]byte{78}, "brewer", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.HeldSlot = 0
	p.Health = 10
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := p.Inventory[player.HotbarStart].SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:healing",
	}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-player.FoodUseDuration("minecraft:potion"))
	if !startJavaFoodUse(p, player.HotbarStart, started) || !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("healing potion did not complete")
	}
	if health, _, _, _ := p.HealthSnapshot(); health != 14 {
		t.Fatalf("health after potion = %.1f, want 14", health)
	}
	if stack := p.HeldItem(); stack.ItemID != "minecraft:glass_bottle" || stack.Count != 1 {
		t.Fatalf("potion remainder = %+v", stack)
	}
}

func TestJavaDrinkablePotionStoresTimedEffect(t *testing.T) {
	p := player.New([16]byte{79}, "brewer", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := p.Inventory[player.HotbarStart].SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:long_swiftness",
	}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-player.FoodUseDuration("minecraft:potion"))
	if !startJavaFoodUse(p, player.HotbarStart, started) || !TickJavaFoodUse(p, nil, nil, time.Now()) {
		t.Fatal("speed potion did not complete")
	}
	effect, ok := p.StatusEffect("speed")
	if !ok || effect.Amplifier != 0 || effect.Duration != 9600 {
		t.Fatalf("stored speed effect = %+v, ok=%v", effect, ok)
	}
}
