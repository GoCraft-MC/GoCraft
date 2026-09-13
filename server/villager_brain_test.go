package server

import (
	"testing"

	corentity "GoCraft/core/entity"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestVillagerBrainRegistryMatchesMojang1214(t *testing.T) {
	if got := len(villagerMemoryModules); got != 30 {
		t.Fatalf("villager memory module count = %d; want 30 for 1.21.4", got)
	}
	if got := len(villagerSensorTypes); got != 9 {
		t.Fatalf("villager sensor count = %d; want 9 for 1.21.4", got)
	}
	wantMemories := map[string]bool{
		"home": true, "job_site": true, "potential_job_site": true, "meeting_point": true,
		"nearest_living_entities": true, "nearest_visible_living_entities": true, "visible_villager_babies": true,
		"nearest_players": true, "nearest_visible_player": true, "nearest_visible_attackable_player": true,
		"nearest_visible_wanted_item": true, "item_pickup_cooldown_ticks": true, "walk_target": true,
		"look_target": true, "interaction_target": true, "breed_target": true, "path": true,
		"doors_to_close": true, "nearest_bed": true, "hurt_by": true, "hurt_by_entity": true,
		"nearest_hostile": true, "secondary_job_site": true, "hiding_place": true, "heard_bell_time": true,
		"cant_reach_walk_target_since": true, "last_slept": true, "last_woken": true,
		"last_worked_at_poi": true, "golem_detected_recently": true,
	}
	for _, memory := range villagerMemoryModules {
		if !wantMemories[memory] {
			t.Fatalf("unexpected villager memory %q", memory)
		}
		delete(wantMemories, memory)
	}
	if len(wantMemories) != 0 {
		t.Fatalf("missing villager memories: %+v", wantMemories)
	}
}

func TestVillagerSchedulesMatchVanilla1214(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	server := &Server{world: world, mobAIs: make(map[int32]*mobAI)}
	adult := corentity.New(100, [16]byte{}, corentity.TypeVillager, 1.5, 64, 1.5)
	adult.HasVillageWorkstation = true
	adult.VillageWorkstation = spatial.BlockPos{X: 2, Y: 64, Z: 2}
	adult.VillageCenter = spatial.BlockPos{X: 4, Y: 64, Z: 4}
	world.SetBlock(2, 64, 2, coreworld.Block{Namespace: "minecraft", Name: "lectern"})
	ai := server.mobAIFor(adult)
	state := server.villagerBrainStateFor(adult)

	adultCases := []struct {
		time int64
		want villagerActivity
	}{
		{10, villagerActivityIdle},
		{1999, villagerActivityIdle},
		{2000, villagerActivityWork},
		{8999, villagerActivityWork},
		{9000, villagerActivityMeet},
		{10999, villagerActivityMeet},
		{11000, villagerActivityIdle},
		{11999, villagerActivityIdle},
		{12000, villagerActivityRest},
		{23999, villagerActivityRest},
	}
	for _, tc := range adultCases {
		server.worldAge = tc.time
		if got := server.selectVillagerActivity(adult, ai, state); got != tc.want {
			t.Fatalf("adult activity at %d = %v; want %v", tc.time, got, tc.want)
		}
	}

	baby := corentity.New(101, [16]byte{}, corentity.TypeVillager, 1.5, 64, 1.5)
	baby.IsBaby = true
	babyAI := server.mobAIFor(baby)
	babyState := server.villagerBrainStateFor(baby)
	babyCases := []struct {
		time int64
		want villagerActivity
	}{
		{10, villagerActivityIdle},
		{2999, villagerActivityIdle},
		{3000, villagerActivityPlay},
		{5999, villagerActivityPlay},
		{6000, villagerActivityIdle},
		{9999, villagerActivityIdle},
		{10000, villagerActivityPlay},
		{11999, villagerActivityPlay},
		{12000, villagerActivityRest},
	}
	for _, tc := range babyCases {
		server.worldAge = tc.time
		if got := server.selectVillagerActivity(baby, babyAI, babyState); got != tc.want {
			t.Fatalf("baby activity at %d = %v; want %v", tc.time, got, tc.want)
		}
	}
}

func TestVillagerBellMemoryActivatesHide(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	server := &Server{world: world, worldAge: 5000, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(102, [16]byte{}, corentity.TypeVillager, 4.5, 64, 4.5)
	world.Entities.Add(villager)
	server.notifyVillagersOfBell(world, spatial.BlockPos{X: 5, Y: 64, Z: 5})
	state := server.villagerBrainStateFor(villager)
	if !state.hasHeardBellTime || state.heardBellTime != 5000 {
		t.Fatalf("HEARD_BELL_TIME = (%v,%d); want (true,5000)", state.hasHeardBellTime, state.heardBellTime)
	}
	if got := server.selectVillagerActivity(villager, server.mobAIFor(villager), state); got != villagerActivityHide {
		t.Fatalf("activity after bell = %v; want HIDE", got)
	}
}

func TestVillagerHostileSensorActivatesPanic(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	server := &Server{world: world, worldAge: 100, mobAIs: make(map[int32]*mobAI)}
	villager := corentity.New(103, [16]byte{}, corentity.TypeVillager, 4.5, 64, 4.5)
	zombie := corentity.New(104, [16]byte{}, corentity.TypeZombie, 8.5, 64, 4.5)
	world.Entities.Add(villager)
	world.Entities.Add(zombie)
	ai := server.mobAIFor(villager)
	state := server.villagerBrainStateFor(villager)
	server.tickVillagerSensors(villager, ai, state)
	if state.nearestHostileID != zombie.EntityID {
		t.Fatalf("NEAREST_HOSTILE = %d; want %d", state.nearestHostileID, zombie.EntityID)
	}
	if got := server.selectVillagerActivity(villager, ai, state); got != villagerActivityPanic {
		t.Fatalf("activity with nearby zombie = %v; want PANIC", got)
	}
}
