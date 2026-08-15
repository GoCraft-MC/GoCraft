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

func TestWindChargeBurstsOnImpactAndKnocksEntitiesUp(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(2, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	charge := corentity.New(10, [16]byte{1}, corentity.TypeWindCharge, 0.5, 65.5, 0.5)
	charge.VX = 1.5
	nearby := corentity.New(11, [16]byte{2}, corentity.TypeCow, 2.5, 64, 1.5)
	w.Entities.Add(charge)
	w.Entities.Add(nearby)
	s := &Server{world: w, game: game.New(), sessions: session.NewManager()}

	if !s.tickProjectile(charge) {
		t.Fatal("wind charge did not burst on solid impact")
	}
	if nearby.VY <= 0 || nearby.VX == 0 && nearby.VZ == 0 {
		t.Fatalf("nearby entity velocity after burst = %.2f/%.2f/%.2f", nearby.VX, nearby.VY, nearby.VZ)
	}
}

func TestEnderPearlImpactTeleportsAndDamagesOwner(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{9}, "pearl-user", player.ClientEditionBedrock)
	p.EntityID = 31
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 65, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	pearl := corentity.New(32, [16]byte{10}, corentity.TypeEnderPearl, 0.5, 65.5, 0.5)
	pearl.OwnerEntityID = p.EntityID
	pearl.VX = 1
	w.SetBlock(1, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	s := &Server{world: w, game: g, sessions: session.NewManager()}
	if !s.tickProjectile(pearl) {
		t.Fatal("ender pearl did not collide")
	}
	if p.Position.X < 1 || p.Health != 15 || p.FallDistance != 0 {
		t.Fatalf("pearl owner after impact: position=%+v health=%.1f fall=%.1f", p.Position, p.Health, p.FallDistance)
	}
}
