package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func TestSleepingVillagerDoesNotRememberWantedItem(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	server := &Server{world: world, worldAge: 20, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(201, [16]byte{}, corentity.TypeVillager, 4.5, 64, 4.5)
	villager.Sleeping = true
	item := corentity.New(202, [16]byte{}, corentity.TypeItem, 5.5, 64, 4.5)
	item.ItemID = "minecraft:bread"
	item.ItemCount = 1
	world.Entities.Add(villager)
	world.Entities.Add(item)
	state := server.villagerBrainStateFor(villager)
	server.tickVillagerSensors(villager, server.mobAIFor(villager), state)
	if state.nearestWantedItem != 0 {
		t.Fatalf("sleeping villager remembered wanted item %d; Paper MC-157464 guard requires no target", state.nearestWantedItem)
	}
}
