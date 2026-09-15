package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestSetSlimeSizeScalesHealth(t *testing.T) {
	for size, wantHP := range map[int32]float32{1: 1, 2: 4, 4: 16} {
		e := corentity.New(1, [16]byte{}, corentity.TypeSlime, 0, 64, 0)
		setSlimeSize(e, size)
		if e.MaxHealth != wantHP || e.Health != wantHP {
			t.Fatalf("size %d health = %.0f/%.0f, want %.0f", size, e.Health, e.MaxHealth, wantHP)
		}
	}
}

func TestSlimeSplitsIntoHalfSizeChildren(t *testing.T) {
	s := newGolemTestServer(t)
	slime := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSlime, 20, 64, 20)
	setSlimeSize(slime, 4)
	s.world.Entities.Add(slime)

	children := s.splitSlimeOnDeath(slime)
	if len(children) < 2 || len(children) > 4 {
		t.Fatalf("split produced %d children, want 2-4", len(children))
	}
	for _, c := range children {
		if c.Type != corentity.TypeSlime {
			t.Fatalf("child type = %s, want slime", c.Type)
		}
		if c.SlimeSize != 2 {
			t.Fatalf("child size = %d, want 2 (half of 4)", c.SlimeSize)
		}
		if c.MaxHealth != 4 {
			t.Fatalf("child health = %.0f, want 4", c.MaxHealth)
		}
	}
}

func TestSmallSlimeDoesNotSplit(t *testing.T) {
	s := newGolemTestServer(t)
	slime := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeSlime, 20, 64, 20)
	setSlimeSize(slime, 1)
	s.world.Entities.Add(slime)

	if children := s.splitSlimeOnDeath(slime); children != nil {
		t.Fatalf("size-1 slime split into %d children, want none", len(children))
	}
}

func TestNaturalCubeMobGetsValidSize(t *testing.T) {
	s := newGolemTestServer(t)
	magma := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeMagmaCube, 20, 64, 20)
	s.sizeNaturalCubeMob(magma)
	switch magma.SlimeSize {
	case 1, 2, 4:
	default:
		t.Fatalf("natural size = %d, want 1/2/4", magma.SlimeSize)
	}
	if magma.MaxHealth != float32(magma.SlimeSize*magma.SlimeSize) {
		t.Fatalf("health %.0f does not match size %d squared", magma.MaxHealth, magma.SlimeSize)
	}
}
