package server

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestSkeletonStrafesInBowRangeInsteadOfStandingStill(t *testing.T) {
	s := newGolemTestServer(t)
	skel := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSkeleton, 20, 64, 20)
	ai := s.mobAIFor(skel)
	target := spatial.Vec3{X: 28, Y: 64, Z: 20} // 8 blocks away, inside bow range

	s.tickSkeletonStrafe(skel, ai, target, 8)

	if skel.VX == 0 && skel.VZ == 0 {
		t.Fatal("skeleton stood still in bow range instead of strafing")
	}
	// 8 blocks is below the back-off threshold (15*0.5=7.5) but above it, so it
	// should be circling, i.e. have a lateral (Z) component toward the target line.
	if math.Abs(skel.VZ) < 1e-6 {
		t.Fatalf("skeleton is not circling the target: v=(%.3f,%.3f)", skel.VX, skel.VZ)
	}
}

func TestSkeletonBacksOffWhenTargetTooClose(t *testing.T) {
	s := newGolemTestServer(t)
	skel := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSkeleton, 20, 64, 20)
	ai := s.mobAIFor(skel)
	target := spatial.Vec3{X: 23, Y: 64, Z: 20} // 3 blocks — below the 7.5 threshold

	s.tickSkeletonStrafe(skel, ai, target, 3)

	if !ai.strafeBackwards {
		t.Fatal("skeleton did not switch to backing away from a too-close target")
	}
	if skel.VX >= 0 {
		t.Fatalf("skeleton should move away from the +X target while backing off: vx=%.3f", skel.VX)
	}
}

func TestSkeletonFleesFromNearbyWolf(t *testing.T) {
	s := newGolemTestServer(t)
	skel := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSkeleton, 20, 64, 20)
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeWolf, 23, 64, 20)
	s.world.Entities.Add(skel)
	s.world.Entities.Add(wolf)
	ai := s.mobAIFor(skel)

	if !s.tickSkeletonAvoidance(skel, ai) {
		t.Fatal("skeleton did not flee the nearby wolf")
	}
	if skel.VX > 0 {
		t.Fatalf("skeleton fled toward the wolf (wolf at +X): vx=%.3f", skel.VX)
	}
}

func TestSkeletonFindsShadeUnderRoof(t *testing.T) {
	s := newGolemTestServer(t)
	// Roof one column over so it becomes sky-occluded (shaded).
	s.world.SetBlock(25, 68, 20, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	skel := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSkeleton, 20, 64, 20)

	shade, ok := s.findShadeNear(skel, 12)
	if !ok {
		t.Fatal("no shade found under the roof")
	}
	if int(math.Floor(shade.X)) != 25 || int(math.Floor(shade.Z)) != 20 {
		t.Fatalf("shade at %+v, want the roofed column ~(25, 20)", shade)
	}
}

func TestSkeletonSeeksShadeInDaylight(t *testing.T) {
	s := newGolemTestServer(t)
	s.simulationDimension = dimensionOverworld
	s.worldAge = 1000 // daytime
	s.world.SetBlock(25, 68, 20, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	skel := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSkeleton, 20, 64, 20)
	s.world.Entities.Add(skel)
	ai := s.mobAIFor(skel)

	if !s.tickSkeletonAvoidance(skel, ai) {
		t.Fatal("exposed skeleton did not flee the sun toward shade")
	}
}
