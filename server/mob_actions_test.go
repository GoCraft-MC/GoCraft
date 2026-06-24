package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	coreworld "GoCraft/core/world"
)

func TestPumpkinZombieActiveTargetAttacksVillager(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	zombie := corentity.New(1, [16]byte{}, corentity.TypeZombie, 1.5, 64, 1.5)
	villager := corentity.New(2, [16]byte{}, corentity.TypeVillager, 2.5, 64, 1.5)
	zombie.OnGround, villager.OnGround = true, true
	world.Entities.Add(zombie)
	world.Entities.Add(villager)
	s := &Server{world: world, game: game.New(), mobAIs: make(map[int32]*mobAI)}

	s.tickHostileMobAI(zombie)
	damage := world.DrainEntityDamage()
	event, ok := damage[villager.EntityID]
	if !ok || event.Amount != 3 {
		t.Fatalf("zombie-villager damage = %+v, present=%v", event, ok)
	}
}

func TestPumpkinIronGolemActiveTargetAttacksZombie(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	golem := corentity.New(3, [16]byte{}, corentity.TypeIronGolem, 1.5, 64, 1.5)
	zombie := corentity.New(4, [16]byte{}, corentity.TypeZombie, 2.5, 64, 1.5)
	golem.OnGround, zombie.OnGround = true, true
	world.Entities.Add(golem)
	world.Entities.Add(zombie)
	s := &Server{world: world, game: game.New(), mobAIs: make(map[int32]*mobAI)}

	s.tickGolemAI(golem)
	damage := world.DrainEntityDamage()
	event, ok := damage[zombie.EntityID]
	if !ok || event.Amount < 7 || event.Amount > 21 {
		t.Fatalf("golem-zombie damage = %+v, present=%v", event, ok)
	}
}

func TestPumpkinTargetLists(t *testing.T) {
	tests := []struct {
		attacker, target corentity.EntityType
		want             bool
	}{
		{corentity.TypeZombie, corentity.TypeVillager, true},
		{corentity.TypeZombie, corentity.TypeIronGolem, true},
		{corentity.TypePillager, corentity.TypeVillager, true},
		{corentity.TypeGuardian, corentity.TypeAxolotl, true},
		{corentity.TypeEnderman, corentity.TypeEndermite, true},
		{corentity.TypeCreeper, corentity.TypeVillager, false},
		{corentity.TypeSkeleton, corentity.TypeVillager, false},
	}
	for _, test := range tests {
		if got := pumpkinMobTargets(test.attacker, test.target); got != test.want {
			t.Errorf("pumpkinMobTargets(%s, %s) = %v, want %v", test.attacker, test.target, got, test.want)
		}
	}
}
