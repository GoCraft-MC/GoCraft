package world

import "testing"

func TestNearestStrongholdUsesPumpkinRings(t *testing.T) {
	generator := NewOverworldGenerator(0)
	x, z, ok := generator.NearestStronghold(0, 0, 100)
	if !ok {
		t.Fatal("expected a stronghold within Pumpkin's 100-chunk search radius")
	}
	if x != -184 || z != -1784 {
		t.Fatalf("nearest seed-zero stronghold = %d,%d; want -184,-1784", x, z)
	}
	if againX, againZ, againOK := generator.NearestStronghold(0, 0, 100); !againOK || againX != x || againZ != z {
		t.Fatalf("cached placement = %d,%d,%v; want %d,%d,true", againX, againZ, againOK, x, z)
	}
}

func TestStrongholdBiomeTag(t *testing.T) {
	if strongholdBiome("minecraft:ocean") || !strongholdBiome("minecraft:plains") {
		t.Fatal("stronghold biome membership does not match the generated Pumpkin tag")
	}
}
