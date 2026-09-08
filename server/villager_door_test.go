package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestVillagerOpensDoorToReachBedAndSleep(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	stone := coreworld.Block{Namespace: "minecraft", Name: "stone"}
	for z := 2; z <= 14; z++ {
		world.SetBlock(7, 64, z, stone)
		world.SetBlock(7, 65, z, stone)
		for _, x := range []int{1, 14} {
			world.SetBlock(x, 64, z, stone)
			world.SetBlock(x, 65, z, stone)
		}
	}
	for x := 1; x <= 14; x++ {
		for _, z := range []int{2, 14} {
			world.SetBlock(x, 64, z, stone)
			world.SetBlock(x, 65, z, stone)
		}
	}
	lower := coreworld.Block{Namespace: "minecraft", Name: "oak_door", Properties: map[string]string{
		"facing": "east", "half": "lower", "open": "false", "powered": "false",
	}}
	upper := lower
	upper.Properties = copyStringMap(lower.Properties)
	upper.Properties["half"] = "upper"
	world.SetBlock(7, 64, 8, lower)
	world.SetBlock(7, 65, 8, upper)
	bedPosition := spatial.BlockPos{X: 9, Y: 64, Z: 7}
	world.SetBlock(9, 64, 7, coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{
		"part": "head", "facing": "east", "occupied": "false",
	}})

	server := &Server{world: world, worldAge: 13000, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(21, [16]byte{}, corentity.TypeVillager, 4.5, 64, 8.5)
	villager.HasVillageHome = true
	villager.VillageBed = bedPosition
	villager.VillageCenter = bedPosition
	villager.OnGround = true
	world.Entities.Add(villager)
	ai := server.mobAIFor(villager)
	opened := false
	for tick := 0; tick < 300 && !villager.Sleeping; tick++ {
		server.tickVillagerDoor(villager, ai)
		previous := villager.Position
		server.tickPassiveMobAI(villager)
		server.tickAuxiliaryMobPhysics(villager, previous)
		opened = opened || world.GetBlock(7, 64, 8).Properties["open"] == "true"
	}
	if !opened {
		t.Fatal("villager never opened the wooden door")
	}
	if !villager.Sleeping {
		t.Fatalf("villager stopped at %+v instead of reaching bed %+v", villager.Position, bedPosition)
	}
	for range villagerDoorCloseTicks + 1 {
		server.tickVillagerDoor(villager, ai)
	}
	if got := world.GetBlock(7, 64, 8).Properties["open"]; got != "false" {
		t.Fatalf("door remained open after villager passed: open=%q", got)
	}
}
