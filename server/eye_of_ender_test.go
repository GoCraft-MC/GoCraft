package server

import (
	"math"
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
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
