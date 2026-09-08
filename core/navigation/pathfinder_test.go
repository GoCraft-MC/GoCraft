package navigation

import (
	"testing"

	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestFindPathRoutesAroundTwoBlockWall(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	for z := 0; z <= 3; z++ {
		world.SetBlock(4, 64, z, coreworld.Block{Namespace: "minecraft", Name: "stone"})
		world.SetBlock(4, 65, z, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	}

	path, reached := FindPath(world, spatial.Vec3{X: 1.5, Y: 64, Z: 1.5}, spatial.Vec3{X: 7.5, Y: 64, Z: 1.5}, 2048)
	if !reached || len(path) == 0 {
		t.Fatalf("path = %+v, reached=%v", path, reached)
	}
	last := path[len(path)-1]
	if last.X != 7.5 || last.Z != 1.5 {
		t.Fatalf("last waypoint = %+v, want goal centre", last)
	}
	for _, waypoint := range path {
		if waypoint.X == 4.5 && waypoint.Z >= 0.5 && waypoint.Z <= 3.5 {
			t.Fatalf("path crossed wall at %+v: %+v", waypoint, path)
		}
	}
}

func TestFindPathUsesOneBlockStep(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	for z := 0; z <= 3; z++ {
		world.SetBlock(2, 64, z, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	}

	path, reached := FindPath(world, spatial.Vec3{X: 1.5, Y: 64, Z: 1.5}, spatial.Vec3{X: 3.5, Y: 64, Z: 1.5}, 512)
	if !reached {
		t.Fatalf("step path not reached: %+v", path)
	}
	foundStep := false
	for _, waypoint := range path {
		if waypoint.X == 2.5 && waypoint.Y == 65 {
			foundStep = true
		}
	}
	if !foundStep {
		t.Fatalf("path did not step onto obstacle: %+v", path)
	}
}

func TestFindPathDoesNotLoadMissingChunks(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)

	path, reached := FindPath(world, spatial.Vec3{X: 1.5, Y: 64, Z: 1.5}, spatial.Vec3{X: 20.5, Y: 64, Z: 1.5}, 512)
	if reached || len(path) == 0 {
		t.Fatalf("unloaded-chunk search = %+v, reached=%v; want useful partial path", path, reached)
	}
	if world.IsChunkLoaded(1, 0) {
		t.Fatal("pathfinder loaded the destination chunk")
	}
}

func TestFindPathOpeningDoorsUsesClosedWoodenDoor(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	stone := coreworld.Block{Namespace: "minecraft", Name: "stone"}
	for z := 2; z <= 14; z++ {
		world.SetBlock(7, 64, z, coreworld.Block{Namespace: "minecraft", Name: "stone"})
		world.SetBlock(7, 65, z, coreworld.Block{Namespace: "minecraft", Name: "stone"})
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
	door := coreworld.Block{Namespace: "minecraft", Name: "oak_door", Properties: map[string]string{"open": "false"}}
	world.SetBlock(7, 64, 8, door)
	world.SetBlock(7, 65, 8, door)
	start := spatial.Vec3{X: 3.5, Y: 64, Z: 8.5}
	goal := spatial.Vec3{X: 11.5, Y: 64, Z: 8.5}
	if path, reached := FindPath(world, start, goal, 2048); reached {
		t.Fatalf("ordinary path unexpectedly crossed a closed door: %+v", path)
	}
	path, reached := FindPathOpeningDoors(world, start, goal, 2048)
	if !reached {
		t.Fatalf("door-opening path did not reach goal: %+v", path)
	}
	foundDoor := false
	for _, waypoint := range path {
		foundDoor = foundDoor || waypoint.X == 7.5 && waypoint.Z == 8.5
	}
	if !foundDoor {
		t.Fatalf("door-opening path omitted door node: %+v", path)
	}
}

func BenchmarkFindPath(b *testing.B) {
	for _, unreachable := range []bool{false, true} {
		name := "reachable"
		if unreachable {
			name = "unreachable"
		}
		b.Run(name, func(b *testing.B) {
			world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer world.Close()
			for cx := int32(-2); cx <= 2; cx++ {
				for cz := int32(-2); cz <= 2; cz++ {
					world.Chunk(cx, cz)
				}
			}
			goal := spatial.Vec3{X: 12.5, Y: 64, Z: 12.5}
			if unreachable {
				goal.Y = 80
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				FindPath(world, spatial.Vec3{X: 1.5, Y: 64, Z: 1.5}, goal, 4096)
			}
		})
	}
}
