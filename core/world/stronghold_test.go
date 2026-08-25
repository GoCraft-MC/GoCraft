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

func TestStrongholdPortalRoomCarvesUndergroundShell(t *testing.T) {
	generator := NewOverworldGenerator(0)
	chunk := &Chunk{X: 0, Z: 0}
	generator.placeStrongholdPortalRoom(chunk, 8, 32, 8)
	if got := chunkBlock(chunk, 8, 32, 8).ResourceLocation(); got != "minecraft:stone_bricks" &&
		got != "minecraft:cracked_stone_bricks" && got != "minecraft:mossy_stone_bricks" {
		t.Fatalf("portal-room floor = %q", got)
	}
	if got := chunkBlock(chunk, 8, 33, 8).ResourceLocation(); got != "minecraft:air" {
		t.Fatalf("portal-room interior = %q, want air", got)
	}
	if got := chunkBlock(chunk, 14, 35, 8).ResourceLocation(); got == "minecraft:air" {
		t.Fatal("portal-room outer wall was not generated")
	}
}
