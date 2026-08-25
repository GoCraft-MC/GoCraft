package world

import (
	"math"
	"testing"
)

func TestLegacyRandomMatchesJava(t *testing.T) {
	if got := newLegacyRandom(0).nextDouble(); math.Abs(got-0.730967787376657) > 1e-15 {
		t.Fatalf("nextDouble = %.16f", got)
	}
	if got := newLegacyRandom(0).nextInt(100); got != 60 {
		t.Fatalf("nextInt(100) = %d, want 60", got)
	}
}
