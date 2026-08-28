package handler

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestBedRespawnFindsSafeSpaceBesideBed(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{"part": "foot", "facing": "south"}})
	w.SetBlock(0, 64, 1, coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{"part": "head", "facing": "south"}})
	p := player.New([16]byte{}, "sleeper", player.ClientEditionJava)
	p.HasSpawnPoint = true
	p.SpawnPoint = spatial.BlockPos{X: 0, Y: 64, Z: 0}

	position, ok := resolveBedRespawn(p, w)
	if !ok {
		t.Fatal("valid bed did not produce a respawn position")
	}
	if int(position.X-0.5) == 0 && int(position.Z-0.5) == 0 {
		t.Fatalf("respawn position remained inside bed: %+v", position)
	}
}

func TestRespawnBootstrapCompletesAfterThreeByThreeArea(t *testing.T) {
	if got := respawnBootstrapCount(12, len(chunkKeysAround(0, 0, 12))); got != 9 {
		t.Fatalf("large-view bootstrap count = %d, want 9", got)
	}
	if got := respawnBootstrapCount(0, 1); got != 1 {
		t.Fatalf("zero-view bootstrap count = %d, want 1", got)
	}
	if got := respawnBootstrapCount(2, 4); got != 4 {
		t.Fatalf("short available bootstrap count = %d, want 4", got)
	}
}

func TestMissingBedFallsBackToWorldSpawn(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	p := player.New([16]byte{}, "sleeper", player.ClientEditionJava)
	p.HasSpawnPoint = true
	p.SpawnPoint = spatial.BlockPos{X: 9, Y: 64, Z: 9}
	if _, ok := resolveBedRespawn(p, w); ok || p.HasSpawnPoint {
		t.Fatal("missing bed remained a valid personal spawn")
	}
}

func TestDeathRespawnMovesFromEndToOverworldBed(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	p := player.New([16]byte{7}, "traveler", player.ClientEditionJava)
	p.Dimension = 2
	p.WorldSpawn = spatial.Vec3{X: 20.5, Y: 65, Z: 20.5}
	p.SpawnPoint = spatial.BlockPos{X: 0, Y: 64, Z: 0}
	p.HasSpawnPoint = true
	p.ApplyDamage(p.MaxHealth, "fell out of the world")
	w.SetBlock(0, 64, 0, coreworld.Block{
		Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{"facing": "north", "part": "foot"},
	})

	respawnPlayerInOverworld(p, w)
	health, food, _, dead := p.HealthSnapshot()
	if dead || health != 20 || food != 20 || p.Dimension != 0 {
		t.Fatalf("respawn state = health %.1f food %d dead %t dimension %d", health, food, dead, p.Dimension)
	}
	if p.Position == p.WorldSpawn {
		t.Fatal("valid bed respawn fell back to world spawn")
	}
}
