package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	coreworld "GoCraft/core/world"
)

func TestFarmerVillagerDiscoversSecondaryFarmlandPOI(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	world.SetBlock(6, 64, 5, coreworld.Block{Namespace: "minecraft", Name: "farmland"})
	server := &Server{world: world, worldAge: 20, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(203, [16]byte{}, corentity.TypeVillager, 5.5, 64, 5.5)
	villager.VillagerProfession = corentity.VillagerProfessionFarmer
	world.Entities.Add(villager)
	state := server.villagerBrainStateFor(villager)
	server.tickVillagerSensors(villager, server.mobAIFor(villager), state)
	if len(state.secondaryJobSites) == 0 {
		t.Fatal("farmer did not populate SECONDARY_JOB_SITE from nearby farmland")
	}
}
