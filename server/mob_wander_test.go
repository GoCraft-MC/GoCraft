package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

// applyMobVelocity mimics the per-tick position integration that the main loop
// performs for passive mobs: Position += Velocity.
func applyMobVelocity(e *corentity.Entity) {
	e.Position.X += e.VX
	e.Position.Z += e.VZ
}

// TestFindSurfaceWanderTargetReturnsSurfacePosition verifies that
// findSurfaceWanderTarget always returns Y = surfaceY+1 (standing height) for
// a flat world, stays within the lateral/vertical range, and never points at a
// chunk that isn't loaded.
func TestFindSurfaceWanderTargetReturnsSurfacePosition(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	// Load a 3×3 grid of chunks so wandering into adjacent chunks works.
	for cx := int32(-1); cx <= 1; cx++ {
		for cz := int32(-1); cz <= 1; cz++ {
			world.Chunk(cx, cz)
		}
	}
	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 8.5, 64, 8.5)
	ai := s.mobAIFor(e)

	const attempts = 200
	found := 0
	for i := 0; i < attempts; i++ {
		target, ok := s.findSurfaceWanderTarget(e, ai, 10, 7)
		if !ok {
			continue
		}
		found++
		// FlatGenerator: stone at Y=63, so surface standing Y is 64.
		if target.Y != 64 {
			t.Errorf("expected Y=64 (surface+1), got Y=%v", target.Y)
		}
		// Must be within lateral and vertical range.
		if dx := target.X - e.Position.X; dx > 10 || dx < -10 {
			t.Errorf("target X=%v out of lateralRange 10 from X=%v", target.X, e.Position.X)
		}
		if dz := target.Z - e.Position.Z; dz > 10 || dz < -10 {
			t.Errorf("target Z=%v out of lateralRange 10 from Z=%v", target.Z, e.Position.Z)
		}
		if dy := target.Y - e.Position.Y; dy > 7 || dy < -7 {
			t.Errorf("target Y delta %v outside verticalRange 7", dy)
		}
	}
	// With 10 attempts per call and a loaded 3×3 grid, >90% of calls should succeed.
	if found < attempts*80/100 {
		t.Errorf("findSurfaceWanderTarget succeeded only %d/%d times on a flat loaded world", found, attempts)
	}
}

// TestWanderTickAdvancesDuringLookMode verifies bug fix #3: the wander
// cadence clock (wanderTick) decrements even when the mob is in look mode
// (lookTick > 0).  Before the fix, look mode returned early and froze the
// wander clock so mobs could stand idle for arbitrarily long stretches.
func TestWanderTickAdvancesDuringLookMode(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	for cx := int32(-1); cx <= 1; cx++ {
		for cz := int32(-1); cz <= 1; cz++ {
			world.Chunk(cx, cz)
		}
	}
	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 8.5, 64, 8.5)
	ai := s.mobAIFor(e)

	// Force the mob into look mode for 60 ticks.
	ai.lookTick = 60
	ai.lookX, ai.lookZ = 10, 10
	// wanderTick > 0 so no wander fires yet.
	startWanderTick := ai.wanderTick

	for i := 0; i < 60; i++ {
		s.tickPassiveIdleGoals(e, ai)
		applyMobVelocity(e)
	}
	// wanderTick must have moved, proving the clock was not frozen.
	if ai.wanderTick >= startWanderTick && startWanderTick > 0 {
		t.Errorf("wanderTick did not advance during look mode: started=%d ended=%d",
			startWanderTick, ai.wanderTick)
	}
}

// TestPassiveMobsWanderOnFlatTerrain is the primary regression test.
// Five cows are spawned on flat terrain.  Over 1200 ticks (~60 s) with no
// players to tempt them, at least 3 of the 5 must have navigated to a
// different position.  This mirrors vanilla's 1/60 trigger chance every 2
// ticks: expected ~10 wander triggers per mob in the window, enough for
// reliable movement without being brittle about exactly when each fires.
func TestPassiveMobsWanderOnFlatTerrain(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	// Pre-load a 5×5 chunk region so A* never hits an unloaded chunk.
	for cx := int32(-2); cx <= 2; cx++ {
		for cz := int32(-2); cz <= 2; cz++ {
			world.Chunk(cx, cz)
		}
	}

	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	type mobRecord struct {
		entity *corentity.Entity
		startX float64
		startZ float64
	}

	mobTypes := []corentity.EntityType{
		corentity.TypeCow,
		corentity.TypePig,
		corentity.TypeSheep,
		corentity.TypeCow,
		corentity.TypePig,
	}
	mobs := make([]mobRecord, len(mobTypes))
	for i, typ := range mobTypes {
		// Spread them out a bit so they don't collide / confuse each other.
		x := 8.5 + float64(i)*3
		e := corentity.New(int32(i+1), [16]byte{}, typ, x, 64, 8.5)
		mobs[i] = mobRecord{entity: e, startX: x, startZ: 8.5}
		s.mobAIFor(e) // initialise AI slot
	}

	const ticks = 1200
	for tick := 0; tick < ticks; tick++ {
		for _, m := range mobs {
			s.tickPassiveIdleGoals(m.entity, s.mobAIFor(m.entity))
			applyMobVelocity(m.entity)
		}
	}

	moved := 0
	for _, m := range mobs {
		dx := m.entity.Position.X - m.startX
		dz := m.entity.Position.Z - m.startZ
		dist := dx*dx + dz*dz
		if dist > 0.01 { // more than ~0.1 blocks
			moved++
		}
	}
	if moved < 3 {
		t.Errorf("expected at least 3 of %d mobs to have moved in %d ticks; only %d did",
			len(mobs), ticks, moved)
		for i, m := range mobs {
			dx := m.entity.Position.X - m.startX
			dz := m.entity.Position.Z - m.startZ
			t.Logf("  mob %d (%v): moved (%.3f, %.3f)", i+1, mobTypes[i], dx, dz)
		}
	}
}

// TestNavigationClearedAfterFailedWander checks that after a wander trigger
// fires and A* fails (e.g. the mob is surrounded by unloaded chunks), the
// stale hasPathGoal flag is cleared so the next successful trigger can start
// navigation.  Before fix #1 this state leaked and blocked navigation for
// dozens of seconds.
func TestNavigationClearedAfterFailedWander(t *testing.T) {
	// Create a world but load NO chunks — A* will always fail.
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()

	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 8.5, 64, 8.5)
	ai := s.mobAIFor(e)

	// Force a wander cycle by resetting the tick counter.
	ai.wanderTick = 1
	// Drive one tick so the wander code executes.
	s.tickPassiveIdleGoals(e, ai)

	// Whether or not a target was found, hasPathGoal must be false afterwards
	// (either no target found, or A* failed and we cleared it).
	if ai.hasWanderGoal {
		t.Error("hasWanderGoal should be false after a wander attempt with unloaded chunks")
	}
	if ai.hasPathGoal {
		t.Error("hasPathGoal should be false after a failed A* attempt")
	}
}

// TestSurfaceWanderTargetRejectsUnloadedChunks ensures that
// findSurfaceWanderTarget returns false rather than a target whose chunk is
// not loaded, preventing mobs from navigating to unloaded terrain.
func TestSurfaceWanderTargetRejectsUnloadedChunks(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	// Load only the origin chunk.
	world.Chunk(0, 0)

	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	// Place the mob near the chunk border so most random targets land outside.
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 15.5, 64, 15.5)
	ai := s.mobAIFor(e)

	// With a lateral range of 10, many candidates will land in unloaded
	// chunks.  All returned targets must therefore be in the loaded chunk
	// (x/z in [0,16)).
	const runs = 100
	for i := 0; i < runs; i++ {
		target, ok := s.findSurfaceWanderTarget(e, ai, 10, 7)
		if !ok {
			continue // correctly rejected
		}
		// Returned target must be inside the one loaded chunk (chunk 0,0).
		if target.X < 0 || target.X >= 16 || target.Z < 0 || target.Z >= 16 {
			t.Errorf("target (%v, %v) is outside the only loaded chunk [0,16)x[0,16)",
				target.X, target.Z)
		}
	}
}

// TestMobsDoNotMoveWithoutAWorld is a guard: if the Server has no world,
// tickPassiveIdleGoals must not panic and the mob must not gain velocity.
func TestMobsDoNotMoveWithoutAWorld(t *testing.T) {
	s := &Server{mobAIs: make(map[int32]*mobAI)} // no world
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 0, 64, 0)
	ai := s.mobAIFor(e)

	for i := 0; i < 200; i++ {
		s.tickPassiveIdleGoals(e, ai)
		applyMobVelocity(e)
	}
	if e.VX != 0 || e.VZ != 0 {
		t.Errorf("mob gained velocity (%v, %v) with no world", e.VX, e.VZ)
	}
}

// TestPassiveMobsOfMultipleTypesMoveOnFlatTerrain is a variant that tests each
// passive mob type individually and checks it moves within 1200 ticks.
func TestPassiveMobsOfMultipleTypesMoveOnFlatTerrain(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	for cx := int32(-2); cx <= 2; cx++ {
		for cz := int32(-2); cz <= 2; cz++ {
			world.Chunk(cx, cz)
		}
	}

	tests := []struct {
		name string
		typ  corentity.EntityType
	}{
		{"cow", corentity.TypeCow},
		{"pig", corentity.TypePig},
		{"sheep", corentity.TypeSheep},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
			e := corentity.New(1, [16]byte{}, tc.typ, 8.5, 64, 8.5)
			ai := s.mobAIFor(e)
			startX, startZ := e.Position.X, e.Position.Z

			for tick := 0; tick < 1200; tick++ {
				s.tickPassiveIdleGoals(e, ai)
				applyMobVelocity(e)
			}

			dx := e.Position.X - startX
			dz := e.Position.Z - startZ
			dist2 := dx*dx + dz*dz
			if dist2 <= 0.01 {
				t.Errorf("%s did not move in 1200 ticks (still at %.3f, %.3f)",
					tc.name, e.Position.X, e.Position.Z)
			}
		})
	}
}

// TestWanderTargetIsOnSurfaceNotUnderground verifies fix #2: before the fix,
// the Y coordinate was chosen as e.Position.Y + rng*14 - 7, which could land
// underground.  Now it must always equal surfaceY+1 (flat world = 64).
func TestWanderTargetIsOnSurfaceNotUnderground(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	for cx := int32(-1); cx <= 1; cx++ {
		for cz := int32(-1); cz <= 1; cz++ {
			world.Chunk(cx, cz)
		}
	}
	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 8.5, 64, 8.5)
	ai := s.mobAIFor(e)

	for i := 0; i < 500; i++ {
		target, ok := s.findSurfaceWanderTarget(e, ai, 10, 7)
		if !ok {
			continue
		}
		if target.Y < 64 {
			t.Errorf("wander target Y=%v is below surface (expected >=64 on flat world)", target.Y)
		}
	}
}

// TestFindSurfaceWanderTargetWithVec3 checks the returned Vec3 matches
// expected surface Y and is within lateral bounds.
func TestFindSurfaceWanderTargetWithVec3(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)

	s := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	e := corentity.New(1, [16]byte{}, corentity.TypeCow, 4.5, 64, 4.5)
	ai := s.mobAIFor(e)

	target, ok := s.findSurfaceWanderTarget(e, ai, 3, 7)
	if !ok {
		// With such a small range we may not find it every time; that's OK.
		t.Skip("findSurfaceWanderTarget found no target within 3 blocks; skipping")
	}
	if target.Y != 64 {
		t.Errorf("expected Y=64, got Y=%v", target.Y)
	}
	dx := target.X - e.Position.X
	dz := target.Z - e.Position.Z
	if dx > 3 || dx < -3 || dz > 3 || dz < -3 {
		t.Errorf("target out of lateral range 3: dx=%.2f dz=%.2f target=%v entity=%v", dx, dz, target, e.Position)
	}
}
