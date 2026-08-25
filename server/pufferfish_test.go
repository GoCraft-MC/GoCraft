package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	"GoCraft/java/session"
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

func TestPufferfishIgnoresCalmTagAndInflatesForHostiles(t *testing.T) {
	fish := corentity.New(2, [16]byte{}, corentity.TypePufferfish, 0, 64, 0)
	cod := corentity.New(3, [16]byte{}, corentity.TypeCod, 1, 64, 0)
	s := &Server{sessions: session.NewManager(), simulationDimension: dimensionOverworld}
	if s.pufferfishThreatNearby(fish, []*corentity.Entity{fish, cod}) {
		t.Fatal("cod from not_scary_for_pufferfish triggered inflation")
	}
	zombie := corentity.New(4, [16]byte{}, corentity.TypeZombie, 1, 64, 0)
	if !s.pufferfishThreatNearby(fish, []*corentity.Entity{fish, zombie}) {
		t.Fatal("nearby zombie did not trigger inflation")
	}
	zombie.Position = spatial.Vec3{X: 3, Y: 64, Z: 0}
	if s.pufferfishThreatNearby(fish, []*corentity.Entity{fish, zombie}) {
		t.Fatal("distant zombie triggered inflation")
	}
}
