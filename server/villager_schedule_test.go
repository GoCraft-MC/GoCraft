package server

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestVillagerHomeCacheFollowsClaim(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)

	server := &Server{world: world, worldAge: 3000, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(31, [16]byte{}, corentity.TypeVillager, 2.5, 64, 2.5)
	ai := server.mobAIFor(villager)
	if !ai.roaming {
		t.Fatal("new villager without HOME unexpectedly started homed")
	}

	villager.HasVillageHome = true
	villager.VillageBed = spatial.BlockPos{X: 8, Y: 64, Z: 8}
	villager.VillageCenter = spatial.BlockPos{X: 10, Y: 64, Z: 12}
	server.tickVillagerDoor(villager, ai)

	if ai.roaming {
		t.Fatal("villager stayed in roaming mode after acquiring HOME")
	}
	if ai.homeX != 10.5 || ai.homeZ != 12.5 {
		t.Fatalf("cached villager home = (%v,%v); want (10.5,12.5)", ai.homeX, ai.homeZ)
	}

	villager.HasVillageHome = false
	server.tickVillagerDoor(villager, ai)
	if !ai.roaming {
		t.Fatal("villager stayed home-bound after losing HOME")
	}
}

func TestVillagerWorkScheduleTargetsWorkstation(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	job := spatial.BlockPos{X: 10, Y: 64, Z: 10}
	world.SetBlock(int(job.X), int(job.Y), int(job.Z), coreworld.Block{Namespace: "minecraft", Name: "lectern"})

	server := &Server{world: world, worldAge: 3000, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(32, [16]byte{}, corentity.TypeVillager, 2.5, 64, 2.5)
	villager.HasVillageWorkstation = true
	villager.VillageWorkstation = job
	ai := server.mobAIFor(villager)

	server.tickVillagerDoor(villager, ai)

	if !ai.hasWanderGoal {
		t.Fatal("WORK activity did not create a walk target for the villager workstation")
	}
	want := spatial.Vec3{X: 10.5, Y: 64, Z: 10.5}
	if ai.wanderTarget != want {
		t.Fatalf("WORK target = %+v; want %+v", ai.wanderTarget, want)
	}
}

func TestVillagerWorkScheduleWalksToWorkstationEndToEnd(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	job := spatial.BlockPos{X: 10, Y: 64, Z: 10}
	world.SetBlock(int(job.X), int(job.Y), int(job.Z), coreworld.Block{Namespace: "minecraft", Name: "lectern"})

	server := &Server{world: world, worldAge: 3000, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(35, [16]byte{}, corentity.TypeVillager, 2.5, 64, 2.5)
	villager.HasVillageWorkstation = true
	villager.VillageWorkstation = job
	villager.OnGround = true
	world.Entities.Add(villager)
	ai := server.mobAIFor(villager)

	target := spatial.Vec3{X: 10.5, Y: 64, Z: 10.5}
	startDistance := math.Hypot(villager.Position.X-target.X, villager.Position.Z-target.Z)
	for tick := 0; tick < 160; tick++ {
		server.tickVillagerDoor(villager, ai)
		previous := villager.Position
		server.tickPassiveMobAI(villager)
		server.tickAuxiliaryMobPhysics(villager, previous)
	}
	endDistance := math.Hypot(villager.Position.X-target.X, villager.Position.Z-target.Z)
	if endDistance >= startDistance-2 {
		t.Fatalf("WORK target did not survive passive AI: start distance=%.2f end distance=%.2f position=%+v", startDistance, endDistance, villager.Position)
	}
	if endDistance > 2.5 {
		t.Fatalf("villager did not reach workstation vicinity: distance=%.2f position=%+v target=%+v", endDistance, villager.Position, target)
	}
}

func TestVillagerMeetScheduleTargetsVillageCenter(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)

	server := &Server{world: world, worldAge: 9500, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(33, [16]byte{}, corentity.TypeVillager, 2.5, 64, 2.5)
	villager.VillageCenter = spatial.BlockPos{X: 8, Y: 64, Z: 8}
	ai := server.mobAIFor(villager)

	server.tickVillagerDoor(villager, ai)

	if !ai.hasWanderGoal {
		t.Fatal("MEET activity did not create a walk target near the village center")
	}
	centerX, centerZ := 8.5, 8.5
	distance := math.Hypot(ai.wanderTarget.X-centerX, ai.wanderTarget.Z-centerZ)
	if distance < 1.9 || distance > 3.1 {
		t.Fatalf("MEET target %+v is %.2f blocks from center; want a small social offset", ai.wanderTarget, distance)
	}
	if ai.wanderTarget.Y != 64 {
		t.Fatalf("MEET target Y = %v; want 64", ai.wanderTarget.Y)
	}
}

func TestVillagerMeetScheduleWalksToMeetingPointEndToEnd(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)

	server := &Server{world: world, worldAge: 9500, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(36, [16]byte{}, corentity.TypeVillager, 1.5, 64, 1.5)
	villager.VillageCenter = spatial.BlockPos{X: 10, Y: 64, Z: 10}
	villager.OnGround = true
	world.Entities.Add(villager)
	ai := server.mobAIFor(villager)

	server.tickVillagerDoor(villager, ai)
	target := ai.wanderTarget
	startDistance := math.Hypot(villager.Position.X-target.X, villager.Position.Z-target.Z)
	for tick := 0; tick < 180; tick++ {
		server.tickVillagerDoor(villager, ai)
		previous := villager.Position
		server.tickPassiveMobAI(villager)
		server.tickAuxiliaryMobPhysics(villager, previous)
	}
	endDistance := math.Hypot(villager.Position.X-target.X, villager.Position.Z-target.Z)
	if endDistance >= startDistance-2 {
		t.Fatalf("MEET target did not survive passive AI: start distance=%.2f end distance=%.2f position=%+v", startDistance, endDistance, villager.Position)
	}
	if endDistance > 3.0 {
		t.Fatalf("villager did not reach meeting-point vicinity: distance=%.2f position=%+v target=%+v", endDistance, villager.Position, target)
	}
}

func TestVillagerIdleAndBabySchedulesDoNotForceAdultPOITargets(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	job := spatial.BlockPos{X: 10, Y: 64, Z: 10}
	world.SetBlock(int(job.X), int(job.Y), int(job.Z), coreworld.Block{Namespace: "minecraft", Name: "lectern"})

	server := &Server{world: world, worldAge: 11500, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(34, [16]byte{}, corentity.TypeVillager, 2.5, 64, 2.5)
	villager.HasVillageWorkstation = true
	villager.VillageWorkstation = job
	villager.VillageCenter = spatial.BlockPos{X: 8, Y: 64, Z: 8}
	ai := server.mobAIFor(villager)
	server.tickVillagerDoor(villager, ai)
	if ai.hasWanderGoal {
		t.Fatalf("IDLE activity unexpectedly forced POI target %+v", ai.wanderTarget)
	}

	server.worldAge = 3000
	villager.IsBaby = true
	server.tickVillagerDoor(villager, ai)
	if ai.hasWanderGoal {
		t.Fatalf("baby villager unexpectedly followed adult WORK target %+v", ai.wanderTarget)
	}
}
