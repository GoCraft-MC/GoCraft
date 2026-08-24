package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func TestPoweredRailStartsAndAcceleratesMinecart(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	s := &Server{world: w}
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "powered_rail", Properties: map[string]string{
		"shape": "east_west", "powered": "true",
	}})
	cart := corentity.New(1, [16]byte{}, corentity.TypeMinecart, 0.5, 64.0625, 0.5)
	cart.Yaw = -90
	s.tickMinecartPhysics(cart)
	if cart.Position.X <= 0.5 || cart.VX <= 0 {
		t.Fatalf("powered cart position=%+v velocity=(%v,%v)", cart.Position, cart.VX, cart.VZ)
	}
	firstSpeed := cart.VX
	s.tickMinecartPhysics(cart)
	if cart.VX <= firstSpeed {
		t.Fatalf("powered rail did not accelerate: %v <= %v", cart.VX, firstSpeed)
	}
}

func TestUnpoweredPoweredRailBrakesMinecart(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	s := &Server{world: w}
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "powered_rail", Properties: map[string]string{
		"shape": "east_west", "powered": "false",
	}})
	cart := corentity.New(1, [16]byte{}, corentity.TypeMinecart, 0.5, 64.0625, 0.5)
	cart.VX = 0.2
	s.tickMinecartPhysics(cart)
	if cart.VX >= 0.2 || cart.VX <= 0 {
		t.Fatalf("braked velocity = %v, want between zero and 0.2", cart.VX)
	}
}
