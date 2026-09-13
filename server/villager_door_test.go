package server

import (
	"math"
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

	// The next serial villager phase mirrors LivingEntity.startSleeping and
	// anchors the canonical entity position on the bed instead of leaving the
	// sleeping pose at the last pathfinding position beside it.
	server.tickVillagerDoor(villager, ai)
	wantX, wantY, wantZ := float64(bedPosition.X)+0.5, float64(bedPosition.Y)+villagerBedSleepYOffset, float64(bedPosition.Z)+0.5
	if math.Abs(villager.Position.X-wantX) > 1e-9 || math.Abs(villager.Position.Y-wantY) > 1e-9 || math.Abs(villager.Position.Z-wantZ) > 1e-9 {
		t.Fatalf("sleeping villager position = %+v; want (%v,%v,%v)", villager.Position, wantX, wantY, wantZ)
	}

	for range villagerDoorCloseTicks + 1 {
		server.tickVillagerDoor(villager, ai)
	}
	if got := world.GetBlock(7, 64, 8).Properties["open"]; got != "false" {
		t.Fatalf("door remained open after villager passed: open=%q", got)
	}
}

func TestSleepingVillagerNormalisesFootClaimToBedHead(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)

	foot := spatial.BlockPos{X: 4, Y: 64, Z: 4}
	head := spatial.BlockPos{X: 5, Y: 64, Z: 4}
	world.SetBlock(int(foot.X), int(foot.Y), int(foot.Z), coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{
		"part": "foot", "facing": "east", "occupied": "true",
	}})
	world.SetBlock(int(head.X), int(head.Y), int(head.Z), coreworld.Block{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{
		"part": "head", "facing": "east", "occupied": "true",
	}})

	server := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(22, [16]byte{}, corentity.TypeVillager, 3.5, 64, 6.5)
	villager.HasVillageHome = true
	villager.VillageBed = foot
	villager.Sleeping = true
	ai := server.mobAIFor(villager)
	ai.path = []spatial.Vec3{{X: 9, Y: 64, Z: 9}}
	ai.pathIndex = 0

	server.tickVillagerDoor(villager, ai)

	if villager.VillageBed != head {
		t.Fatalf("VillageBed = %+v; want canonical head %+v", villager.VillageBed, head)
	}
	want := spatial.Vec3{X: float64(head.X) + 0.5, Y: float64(head.Y) + villagerBedSleepYOffset, Z: float64(head.Z) + 0.5}
	if villager.Position != want {
		t.Fatalf("sleeping villager position = %+v; want %+v", villager.Position, want)
	}
	if villager.VX != 0 || villager.VY != 0 || villager.VZ != 0 {
		t.Fatalf("sleeping villager retained velocity (%v,%v,%v)", villager.VX, villager.VY, villager.VZ)
	}
	if len(ai.path) != 0 || ai.pathIndex != 0 {
		t.Fatalf("sleeping villager retained navigation path: len=%d index=%d", len(ai.path), ai.pathIndex)
	}
}
