package world

import (
	"math/bits"
	"testing"

	coreworld "GoCraft/core/world"
)

func TestBuildSkyLightDarkensUndergroundAndLightsSky(t *testing.T) {
	chunk := (&coreworld.FlatGenerator{}).Generate(0, 0)
	skyMask, emptyMask, arrays := buildSkyLight(chunk)
	const all = uint64((1 << 26) - 1)
	if uint64(skyMask|emptyMask) != all || skyMask&emptyMask != 0 {
		t.Fatalf("sky masks overlap or omit sections: sky=%026b empty=%026b", skyMask, emptyMask)
	}
	if got := bits.OnesCount64(uint64(skyMask)); got != len(arrays) {
		t.Fatalf("sky arrays=%d, mask bits=%d", len(arrays), got)
	}
	// Y=0 is world section 4, represented by light-mask bit 5.
	if emptyMask&(int64(1)<<5) == 0 {
		t.Fatalf("underground section is not marked empty: %026b", emptyMask)
	}
	// Y=80 is section 9, represented by bit 10, and must see the sky.
	if skyMask&(int64(1)<<10) == 0 {
		t.Fatalf("above-ground section has no sky data: %026b", skyMask)
	}
	if len(arrays) >= 26 {
		t.Fatalf("sent %d sky arrays, want underground zero sections omitted", len(arrays))
	}
}
