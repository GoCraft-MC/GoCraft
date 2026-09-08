package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestSplashPotionImpactAppliesDistanceScaledEffect(t *testing.T) {
	g := game.New()
	near := player.New([16]byte{51}, "near", player.ClientEditionJava)
	near.Position = spatial.Vec3{X: 0, Y: 64, Z: 0}
	far := player.New([16]byte{52}, "far", player.ClientEditionBedrock)
	far.Position = spatial.Vec3{X: 5, Y: 64, Z: 0}
	if err := g.AddPlayer(near); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPlayer(far); err != nil {
		t.Fatal(err)
	}
	stack := player.ItemStack{ItemID: "minecraft:splash_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:poison"}); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, sessions: session.NewManager(), simulationDimension: dimensionOverworld}
	s.resolveProjectileImpact(&corentity.Entity{Type: corentity.TypePotion, ProjectileItem: stack}, spatial.Vec3{Y: 64.9})

	effect, ok := near.StatusEffect("poison")
	if !ok || effect.Duration != 900 {
		t.Fatalf("near poison = %+v, ok=%v", effect, ok)
	}
	if _, ok := far.StatusEffect("poison"); ok {
		t.Fatal("out-of-range player received splash effect")
	}
}

func TestSplashPotionImpactAppliesInstantHealing(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{53}, "hurt", player.ClientEditionBedrock)
	p.Health = 10
	p.Position = spatial.Vec3{X: 2, Y: 64, Z: 2}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	stack := player.ItemStack{ItemID: "minecraft:splash_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:healing"}); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, sessions: session.NewManager(), simulationDimension: dimensionOverworld}
	s.applySplashPotion(stack, spatial.Vec3{X: 2, Y: 64.9, Z: 2})
	if health, _, _, _ := p.HealthSnapshot(); health != 14 {
		t.Fatalf("health after splash potion = %.1f, want 14", health)
	}
}

func TestLingeringSplashUsesQuarterDuration(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{54}, "lingering", player.ClientEditionJava)
	p.Position = spatial.Vec3{Y: 64}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	stack := player.ItemStack{ItemID: "minecraft:lingering_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:poison"}); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, sessions: session.NewManager(), simulationDimension: dimensionOverworld}
	s.applySplashPotionScaled(stack, spatial.Vec3{Y: 64.9}, 0.25)
	if effect, ok := p.StatusEffect("poison"); !ok || effect.Duration != 225 {
		t.Fatalf("lingering splash poison = %+v, ok=%v", effect, ok)
	}
}

func TestLingeringPotionImpactSpawnsCloud(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = w.Close() })
	stack := player.ItemStack{ItemID: "minecraft:lingering_potion", Count: 1}
	if err := stack.SetComponent("potion_contents", map[string]string{"potion": "minecraft:poison"}); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: game.New(), world: w, sessions: session.NewManager(), simulationDimension: dimensionOverworld}
	s.resolveProjectileImpact(&corentity.Entity{Type: corentity.TypePotion, ProjectileItem: stack}, spatial.Vec3{X: 2, Y: 65, Z: 3})
	entities := w.Entities.Snapshot()
	if len(entities) != 1 || entities[0].Type != corentity.TypeAreaEffectCloud {
		t.Fatalf("spawned entities = %+v", entities)
	}
	cloud := entities[0]
	if cloud.Position != (spatial.Vec3{X: 2, Y: 65, Z: 3}) || cloud.ProjectileItem.Components == "" {
		t.Fatalf("cloud payload = %+v", cloud)
	}
}
