package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestWolfBegsAtPlayerHoldingWolfFood(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{5}, "beggee", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 20, Y: 64, Z: 20}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bone", Count: 1}
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{9}, corentity.TypeWolf, 22, 64, 20)
	s.world.Entities.Add(wolf)

	s.tickWolfBegging(wolf)
	if !wolf.WolfBegging {
		t.Fatal("wolf did not beg at a player holding a bone")
	}

	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:dirt", Count: 1}
	s.tickWolfBegging(wolf)
	if wolf.WolfBegging {
		t.Fatal("wolf kept begging at a player holding a non-food item")
	}
}

func TestWolfDoesNotBegAtDistantPlayer(t *testing.T) {
	s := newGolemTestServer(t)
	p := player.New([16]byte{5}, "beggee", player.ClientEditionJava)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 50, Y: 64, Z: 20} // >8 blocks away
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	p.HeldSlot = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bone", Count: 1}
	wolf := corentity.New(s.game.NextEntityID(), [16]byte{9}, corentity.TypeWolf, 22, 64, 20)
	s.world.Entities.Add(wolf)

	s.tickWolfBegging(wolf)
	if wolf.WolfBegging {
		t.Fatal("wolf begged at a player more than 8 blocks away")
	}
}
