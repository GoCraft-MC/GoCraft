package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestPlayerBlockingHitFromDetectsRaisedShieldFacingAttacker(t *testing.T) {
	p := player.New([16]byte{1}, "guard", player.ClientEditionBedrock)
	p.Position = spatial.Vec3{X: 20, Y: 64, Z: 20}
	p.Inventory[player.OffhandSlot] = player.ItemStack{ItemID: "minecraft:shield", Count: 1}
	p.UsingItemID = "minecraft:shield"
	p.Rotation.Yaw = -90 // facing +X, toward an attacker on that side

	if !playerBlockingHitFrom(p, 23, 20) {
		t.Fatal("shield facing the attacker should block")
	}
	// Facing away: no block.
	p.Rotation.Yaw = 90
	if playerBlockingHitFrom(p, 23, 20) {
		t.Fatal("shield facing away should not block")
	}
	// No shield in hand: no block.
	p.Rotation.Yaw = -90
	p.Inventory[player.OffhandSlot] = player.ItemStack{}
	if playerBlockingHitFrom(p, 23, 20) {
		t.Fatal("without a shield there is no block")
	}
}

func TestStunnedRavagerIsImmobile(t *testing.T) {
	s := newGolemTestServer(t)
	target := newPhantomTarget(s.simulationDimension) // reuse the survival-player session helper
	rav := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeRavager, 20, 64, 20)
	rav.VX, rav.VZ = 0.5, 0.5
	s.world.Entities.Add(rav)
	ai := s.mobAIFor(rav)
	parityState(rav).ravagerStunTicks = 40

	s.tickRavagerCombat(rav, ai, target, 5.0, true)
	if rav.VX != 0 || rav.VZ != 0 {
		t.Fatalf("stunned ravager kept moving: vx=%.3f vz=%.3f", rav.VX, rav.VZ)
	}
}
