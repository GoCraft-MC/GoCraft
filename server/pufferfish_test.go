package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestPufferfishInflatesAndDeflatesInThreeStages(t *testing.T) {
	fish := corentity.New(1, [16]byte{}, corentity.TypePufferfish, 0, 64, 0)
	updatePufferState(fish, true)
	if fish.PufferState != 1 {
		t.Fatalf("initial threatened state = %d, want 1", fish.PufferState)
	}
	for tick := 0; tick < 41; tick++ {
		updatePufferState(fish, true)
	}
	if fish.PufferState != 2 {
		t.Fatalf("inflated state = %d, want 2", fish.PufferState)
	}
	for tick := 0; tick < 61; tick++ {
		updatePufferState(fish, false)
	}
	if fish.PufferState != 1 {
		t.Fatalf("first deflation state = %d, want 1", fish.PufferState)
	}
	for tick := 0; tick < 40; tick++ {
		updatePufferState(fish, false)
	}
	if fish.PufferState != 0 {
		t.Fatalf("final deflation state = %d, want 0", fish.PufferState)
	}
}
