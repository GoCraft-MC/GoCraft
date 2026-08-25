package server

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestEyeOfEnderUsesPumpkinHomingVelocity(t *testing.T) {
	eye := corentity.New(1, [16]byte{}, corentity.TypeEyeOfEnder, 0, 65, 0)
	eye.VX, eye.VY = 0.9, 0.15
	eye.EyeTarget = spatial.Vec3{X: 12, Y: 73, Z: 0}
	eye.HasEyeTarget = true
	if tickEyeOfEnder(eye) {
		t.Fatal("new eye expired")
	}
	if math.Abs(eye.Position.X-0.9) > 1e-9 || math.Abs(eye.Position.Y-65.15) > 1e-9 {
		t.Fatalf("position = %+v", eye.Position)
	}
	if math.Abs(eye.VX-0.9255) > 1e-9 || math.Abs(eye.VY-0.16275) > 1e-9 || math.Abs(eye.VZ) > 1e-9 {
		t.Fatalf("velocity = %f,%f,%f", eye.VX, eye.VY, eye.VZ)
	}
}

func TestSurvivingEyeDropsItsCanonicalItem(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	s := &Server{game: game.New(), world: w, sessions: session.NewManager()}
	eye := corentity.New(10, [16]byte{}, corentity.TypeEyeOfEnder, 0, 65, 0)
	eye.AgeTicks = 81
	eye.EyeSurvives = true
	if !s.tickProjectile(eye) {
		t.Fatal("expired eye was retained")
	}
	drops := w.Entities.Snapshot()
	if len(drops) != 1 || drops[0].Type != corentity.TypeItem || drops[0].ItemID != "minecraft:ender_eye" {
		t.Fatalf("eye drops = %+v", drops)
	}
}

func TestEyeOfEnderExpiresAfterPumpkinLifetime(t *testing.T) {
	eye := corentity.New(2, [16]byte{}, corentity.TypeEyeOfEnder, 0, 65, 0)
	eye.AgeTicks = 80
	if tickEyeOfEnder(eye) {
		t.Fatal("eye expired at tick 80")
	}
	eye.AgeTicks = 81
	if !tickEyeOfEnder(eye) {
		t.Fatal("eye survived beyond tick 80")
	}
}
