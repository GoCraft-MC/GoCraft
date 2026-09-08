package server

import (
	"testing"
	"time"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
)

func TestBedrockDrinkablePotionAppliesPayloadAndReturnsBottle(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{47}, "bedrock-brewer", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Health = 10
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := p.Inventory[player.HotbarStart].SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:healing",
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}
	s.applyBedrockStartUseItem(intent.StartUseItemIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	p.UsingItemSince = time.Now().Add(-player.FoodUseDuration("minecraft:potion"))
	s.tickBedrockItemUse()
	if health, _, _, _ := p.HealthSnapshot(); health != 14 {
		t.Fatalf("health after potion = %.1f, want 14", health)
	}
	if stack := p.HeldItem(); stack.ItemID != "minecraft:glass_bottle" || stack.Count != 1 {
		t.Fatalf("potion remainder = %+v", stack)
	}
}

func TestBedrockDrinkablePotionStoresTimedEffect(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{48}, "bedrock-brewer", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:potion", Count: 1}
	if err := p.Inventory[player.HotbarStart].SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:long_swiftness",
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}
	s.applyBedrockStartUseItem(intent.StartUseItemIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	p.UsingItemSince = time.Now().Add(-player.FoodUseDuration("minecraft:potion"))
	s.tickBedrockItemUse()
	effect, ok := p.StatusEffect("speed")
	if !ok || effect.Amplifier != 0 || effect.Duration != 9600 {
		t.Fatalf("stored speed effect = %+v, ok=%v", effect, ok)
	}
}

func TestBedrockSuspiciousStewAppliesStackEffect(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{49}, "bedrock-stew-user", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:suspicious_stew", Count: 1}
	if err := p.Inventory[player.HotbarStart].SetComponent("suspicious_stew_effects", []player.StatusEffect{
		{ID: "minecraft:night_vision", Duration: 100},
	}); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}
	s.applyBedrockStartUseItem(intent.StartUseItemIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	p.UsingItemSince = time.Now().Add(-player.FoodUseDuration("minecraft:suspicious_stew"))
	s.tickBedrockItemUse()
	if effect, ok := p.StatusEffect("night_vision"); !ok || effect.Duration != 100 {
		t.Fatalf("stew effect = %+v, ok=%v", effect, ok)
	}
	if stack := p.HeldItem(); stack.ItemID != "minecraft:bowl" {
		t.Fatalf("stew remainder = %+v", stack)
	}
}
