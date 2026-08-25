package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestWetEndermanTeleportsToDryLoadedGround(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	for cx := int32(-2); cx <= 2; cx++ {
		for cz := int32(-2); cz <= 2; cz++ {
			w.Chunk(cx, cz)
		}
	}
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "water", Properties: map[string]string{"level": "0"}})
	enderman := corentity.New(1, [16]byte{}, corentity.TypeEnderman, 0.5, 64, 0.5)
	s := &Server{world: w, sessions: session.NewManager(), simulationDimension: dimensionOverworld}
	for s.worldAge = 0; ; s.worldAge++ {
		roll := uint64(enderman.EntityID)*0x9e3779b97f4a7c15 ^ uint64(s.worldAge)*0xbf58476d1ce4e5b9
		if roll%10 != 0 {
			break
		}
	}
	hurt := []*corentity.Entity{}
	s.tickEndermanWater(enderman, &hurt)
	if len(hurt) != 1 || enderman.Health != enderman.MaxHealth-1 {
		t.Fatalf("water damage: hurt=%d health=%.1f", len(hurt), enderman.Health)
	}
	if enderman.Position.X == 0.5 && enderman.Position.Z == 0.5 {
		t.Fatal("wet enderman did not teleport")
	}
	if w.TouchesWater(enderman.Position.X, enderman.Position.Y, enderman.Position.Z) {
		t.Fatalf("enderman teleported into water at %+v", enderman.Position)
	}
}
