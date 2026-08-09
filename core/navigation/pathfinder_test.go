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
