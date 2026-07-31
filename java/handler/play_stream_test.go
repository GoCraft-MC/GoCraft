package handler

import "testing"

func TestChunkKeysAroundAreCentreFirstAndUnique(t *testing.T) {
	keys := chunkKeysAround(12, -7, 4)
	if got, want := len(keys), 81; got != want {
		t.Fatalf("keys=%d, want %d", got, want)
	}
	if keys[0] != [2]int32{12, -7} {
		t.Fatalf("first key=%v, want centre", keys[0])
	}
	seen := make(map[[2]int32]struct{}, len(keys))
	lastRing := int32(-1)
	for _, key := range keys {
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate key %v", key)
		}
		seen[key] = struct{}{}
		ring := abs32(key[0] - 12)
		if zRing := abs32(key[1] + 7); zRing > ring {
			ring = zRing
		}
		if ring < lastRing {
			t.Fatalf("ring order moved from %d back to %d", lastRing, ring)
		}
		lastRing = ring
	}
}
