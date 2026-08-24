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
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "powered_rail", Properties: map[string]string{"shape": "east_west", "powered": "true"}})
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
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "powered_rail", Properties: map[string]string{"shape": "east_west", "powered": "false"}})
	cart := corentity.New(1, [16]byte{}, corentity.TypeMinecart, 0.5, 64.0625, 0.5)
	cart.VX = 0.2
	s.tickMinecartPhysics(cart)
	if cart.VX >= 0.2 || cart.VX <= 0 {
		t.Fatalf("braked velocity = %v, want between zero and 0.2", cart.VX)
	}
}

func TestDetectorAndActivatorRailsReactToMinecart(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	s := &Server{world: w}
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "detector_rail", Properties: map[string]string{
		"shape": "east_west", "powered": "false",
	}})
	cart := corentity.New(1, [16]byte{}, corentity.TypeTNTMinecart, 0.5, 64.0625, 0.5)
	s.tickMinecartPhysics(cart)
	if w.GetBlock(0, 64, 0).Properties["powered"] != "true" || !cart.MinecartOnDetector {
		t.Fatal("detector rail did not activate")
	}
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "activator_rail", Properties: map[string]string{
		"shape": "east_west", "powered": "true",
	}})
	cart.Position.X = 1.5
	s.tickMinecartPhysics(cart)
	if w.GetBlock(0, 64, 0).Properties["powered"] != "false" || cart.FuseTicks != 80 {
		t.Fatalf("detector/fuse states = %s/%d", w.GetBlock(0, 64, 0).Properties["powered"], cart.FuseTicks)
	}
}

func TestTNTMinecartFuseCountsDown(t *testing.T) {
	cart := corentity.New(1, [16]byte{}, corentity.TypeTNTMinecart, 0, 64, 0)
	cart.FuseTicks = 80
	s := &Server{}
	if s.tickTNTMinecartFuse(cart) || cart.FuseTicks != 79 {
		t.Fatalf("first fuse tick exploded=%v fuse=%d", false, cart.FuseTicks)
	}
}

func TestMinecartsExchangeVelocityOnCollision(t *testing.T) {
	left := corentity.New(1, [16]byte{}, corentity.TypeMinecart, 0, 64, 0)
	right := corentity.New(2, [16]byte{}, corentity.TypeMinecart, 0.7, 64, 0)
	left.VX, right.VX = 0.2, -0.1
	tickMinecartCollisions(left, []*corentity.Entity{left, right})
	if left.VX >= 0.2 || right.VX <= -0.1 {
		t.Fatalf("collision velocities left=%v right=%v", left.VX, right.VX)
	}
}
