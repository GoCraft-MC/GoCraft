package handler

import (
	"testing"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
	"GoCraft/java/world/anvil"
)

func TestPlacedBedSurvivesWorldReloadWithBothBlockEntities(t *testing.T) {
	directory := t.TempDir()
	storage, err := anvil.NewStorage(directory)
	if err != nil {
		t.Fatal(err)
	}
	w := coreworld.New(&coreworld.FlatGenerator{}, storage, false)
	w.SetBlock(3, 63, 3, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	w.SetBlock(3, 63, 4, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	p := player.New([16]byte{4}, "sleeper", player.ClientEditionJava)
	p.Rotation.Yaw = 0 // south
	if !placeBedBlock(p, 3, 64, 3, "minecraft:red_bed", w, session.NewManager()) {
		t.Fatal("bed placement failed")
	}
	if head := w.GetBlock(3, 64, 4); head.ResourceLocation() != "minecraft:red_bed" || head.Properties["part"] != "head" {
		t.Fatalf("bed head before save = %+v", head)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	reloadedStorage, err := anvil.NewStorage(directory)
	if err != nil {
		t.Fatal(err)
	}
	reloaded := coreworld.New(&coreworld.FlatGenerator{}, reloadedStorage, false)
	defer reloaded.Close()
	if foot := reloaded.GetBlock(3, 64, 3); foot.ResourceLocation() != "minecraft:red_bed" || foot.Properties["part"] != "foot" {
		t.Fatalf("reloaded bed foot = %+v", foot)
	}
	if head := reloaded.GetBlock(3, 64, 4); head.ResourceLocation() != "minecraft:red_bed" || head.Properties["part"] != "head" {
		t.Fatalf("reloaded bed head = %+v", head)
	}
	entities := reloaded.Chunk(0, 0).BlockEntities
	found := map[[3]int]bool{}
	for _, entity := range entities {
		if entity.Type == "minecraft:bed" {
			found[[3]int{entity.X, entity.Y, entity.Z}] = true
		}
	}
	if !found[[3]int{3, 64, 3}] || !found[[3]int{3, 64, 4}] {
		t.Fatalf("reloaded bed block entities = %+v", entities)
	}
}
