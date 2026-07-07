package server

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestBedrockEnteringWaterCancelsAccumulatedFallDamage(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	w.SetBlock(2, 70, 5, coreworld.Block{Namespace: "minecraft", Name: "water", Properties: map[string]string{"level": "0"}})
	p := player.New([16]byte{1}, "diver", player.ClientEditionBedrock)
	p.Position = spatial.Vec3{X: 2.5, Y: 70, Z: 5.5}
	p.OnGround = true
	p.FallDistance = 20
	s := &Server{world: w}

	s.applyBedrockMovementDamage(p, 73, false)
	if p.FallDistance != 0 || p.Health != p.MaxHealth {
		t.Fatalf("water landing state: fall=%.1f health=%.1f", p.FallDistance, p.Health)
	}
}
