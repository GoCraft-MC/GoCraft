package handler

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestUseWindChargeSpawnsProjectileAndConsumesItem(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	p := player.New([16]byte{3}, "thrower", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.EntityID = 7
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:wind_charge", Count: 2}
	if !UseWindCharge(p, w, session.NewManager(), nil, func() int32 { return 42 }) {
		t.Fatal("wind charge use was rejected")
	}
	entities := w.Entities.Snapshot()
	if len(entities) != 1 || entities[0].Type != corentity.TypeWindCharge || entities[0].OwnerEntityID != p.EntityID {
		t.Fatalf("spawned entities = %+v", entities)
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("remaining wind charges = %d, want 1", got)
	}
}

func TestUseThrowableSpawnsCanonicalEggAndConsumesOne(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	p := player.New([16]byte{4}, "thrower", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.EntityID = 8
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:egg", Count: 3}
	if !UseThrowable(p, w, session.NewManager(), nil, func() int32 { return 43 }) {
		t.Fatal("egg use was rejected")
	}
	entities := w.Entities.Snapshot()
	if len(entities) != 1 || entities[0].Type != corentity.TypeEgg || entities[0].OwnerEntityID != 8 {
		t.Fatalf("spawned throwable = %+v", entities)
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 2 {
		t.Fatalf("remaining eggs = %d, want 2", got)
	}
}

func TestUseThrowablePreservesSplashPotionPayload(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	p := player.New([16]byte{5}, "brewer", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.EntityID = 9
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:splash_potion", Count: 2}
	if err := p.Inventory[player.HotbarStart].SetComponent("potion_contents", map[string]string{
		"potion": "minecraft:poison",
	}); err != nil {
		t.Fatal(err)
	}
	if !UseThrowable(p, w, session.NewManager(), nil, func() int32 { return 44 }) {
		t.Fatal("splash potion use was rejected")
	}
	projectile, ok := w.Entities.Get(44)
	if !ok || projectile.Type != corentity.TypePotion || projectile.OwnerEntityID != 9 {
		t.Fatalf("spawned potion = %+v, exists=%v", projectile, ok)
	}
	outcome, ok := player.PotionOutcomeFor(projectile.ProjectileItem)
	if !ok || len(outcome.Effects) != 1 || outcome.Effects[0].ID != "minecraft:poison" {
		t.Fatalf("projectile outcome = %+v, ok=%v", outcome, ok)
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("remaining potions = %d, want 1", got)
	}
}
