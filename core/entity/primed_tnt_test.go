package entity

import "testing"

func TestNewPrimedTNTHasVanillaFuseAndMotion(t *testing.T) {
	tnt := NewPrimedTNT(14, [16]byte{1}, 2.5, 64, -3.5)
	if tnt.Type != TypePrimedTNT || tnt.FuseTicks != 80 || tnt.VY != 0.2 {
		t.Fatalf("primed TNT = %+v", tnt)
	}
	if tnt.Position.X != 2.5 || tnt.Position.Y != 64 || tnt.Position.Z != -3.5 {
		t.Fatalf("primed TNT position = %+v", tnt.Position)
	}
}
