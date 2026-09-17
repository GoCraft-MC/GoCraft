package server

import (
	"testing"

	corentity "GoCraft/core/entity"
)

func TestZoglinTargetsNearbyAnimalsAndMonsters(t *testing.T) {
	s := newGolemTestServer(t)
	zoglin := corentity.New(s.game.NextEntityID(), [16]byte{1}, corentity.TypeZoglin, 20, 64, 20)
	cow := corentity.New(s.game.NextEntityID(), [16]byte{2}, corentity.TypeCow, 22, 64, 20)
	s.world.Entities.Add(zoglin)
	s.world.Entities.Add(cow)

	if got := s.closestPumpkinEntityTarget(zoglin, 16); got == nil || got.EntityID != cow.EntityID {
		t.Fatalf("zoglin did not target the nearby cow: %+v", got)
	}
	// Also hostile toward other monsters.
	if !pumpkinMobTargets(corentity.TypeZoglin, corentity.TypeSkeleton) {
		t.Fatal("zoglin should be hostile to skeletons")
	}
	if !pumpkinMobTargets(corentity.TypeZoglin, corentity.TypeIronGolem) {
		t.Fatal("zoglin should be hostile to iron golems")
	}
}

func TestZoglinIgnoresZoglinsAndCreepers(t *testing.T) {
	if pumpkinMobTargets(corentity.TypeZoglin, corentity.TypeZoglin) {
		t.Fatal("zoglin must not target other zoglins")
	}
	if pumpkinMobTargets(corentity.TypeZoglin, corentity.TypeCreeper) {
		t.Fatal("zoglin must not target creepers")
	}
}

func TestZoglinDoesNotTargetNonLivingEntities(t *testing.T) {
	if pumpkinMobTargets(corentity.TypeZoglin, corentity.TypeArrow) {
		t.Fatal("zoglin should not target projectiles")
	}
	if pumpkinMobTargets(corentity.TypeZoglin, corentity.TypeItem) {
		t.Fatal("zoglin should not target dropped items")
	}
}
