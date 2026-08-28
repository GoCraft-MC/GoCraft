package experience

import (
	"testing"

	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestRoundToOrbSizeMatchesPumpkin(t *testing.T) {
	tests := map[int32]int32{0: 0, 1: 1, 2: 1, 3: 3, 6: 3, 7: 7, 16: 7, 17: 17, 2476: 1237, 2477: 2477}
	for value, want := range tests {
		if got := RoundToOrbSize(value); got != want {
			t.Fatalf("RoundToOrbSize(%d) = %d, want %d", value, got, want)
		}
	}
}

func TestSpawnOrbsPreservesTotal(t *testing.T) {
	world := coreworld.New(nil, nil, false)
	next := int32(0)
	orbs := SpawnOrbs(world, func() int32 { next++; return next }, spatial.Vec3{X: 1, Y: 2, Z: 3}, 2500)
	var total int32
	for _, orb := range orbs {
		total += orb.ExperienceAmount
	}
	if total != 2500 || len(orbs) != 4 {
		t.Fatalf("spawned %d orbs carrying %d XP, want 4 carrying 2500", len(orbs), total)
	}
}
