package server

import (
	corentity "GoCraft/core/entity"
	"testing"
)

func TestSnowballDamageIsZeroExceptBlaze(t *testing.T) {
	snowball := corentity.New(1, [16]byte{}, corentity.TypeSnowball, 0, 0, 0)
	snowball.ProjectileDamage = 99 // generic/default damage must never leak through.
	pig := corentity.New(2, [16]byte{}, corentity.TypePig, 0, 0, 0)
	blaze := corentity.New(3, [16]byte{}, corentity.TypeBlaze, 0, 0, 0)
	if got := projectileDamageAgainst(snowball, pig); got != 0 {
		t.Fatalf("snowball->pig=%v, want 0", got)
	}
	if got := projectileDamageAgainst(snowball, blaze); got != 3 {
		t.Fatalf("snowball->blaze=%v, want 3", got)
	}
}
