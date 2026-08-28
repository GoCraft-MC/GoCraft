package server

import (
	"reflect"
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
)

func TestPlayerDataStoreRoundTrip(t *testing.T) {
	store := newPlayerDataStore(t.TempDir())
	uuid := [16]byte{0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe, 1, 2, 3, 4, 5, 6, 7, 8}
	source := player.New(uuid, "BedrockPlayer", player.ClientEditionBedrock)
	source.Position = spatial.Vec3{X: 123.25, Y: 71, Z: -456.75}
	source.Rotation = spatial.Rotation{Yaw: 127.5, Pitch: -32}
	source.GameMode = player.GameModeSurvival
	source.Health = 17.5
	source.Food = 12
	source.Saturation = 3.5
	source.Exhaustion = 1.25
	source.HeldSlot = 4
	source.SpawnPoint = spatial.BlockPos{X: 91, Y: 65, Z: -12}
	source.HasSpawnPoint = true
	source.Dimension = dimensionEnd
	source.Inventory[5] = player.ItemStack{ItemID: "minecraft:diamond_helmet", Count: 1, Damage: 23}
	source.Inventory[24] = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 48}
	source.Inventory[player.HotbarStart+4] = player.ItemStack{ItemID: "minecraft:diamond_pickaxe", Count: 1, Damage: 57}
	source.Inventory[player.OffhandSlot] = player.ItemStack{ItemID: "minecraft:shield", Count: 1, Damage: 4}
	source.EnderChestInventory[7] = player.ItemStack{ItemID: "minecraft:nether_star", Count: 2}

	// Saving twice exercises replacement of an existing file, which is
	// especially important on Windows where the server commonly runs.
	if err := store.save(uuid, snapshotPlayerData(source)); err != nil {
		t.Fatalf("first save: %v", err)
	}
	source.Inventory[24].Count = 47
	if err := store.save(uuid, snapshotPlayerData(source)); err != nil {
		t.Fatalf("replacement save: %v", err)
	}

	saved, found, err := store.load(uuid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !found {
		t.Fatal("saved player data was not found")
	}
	restored := player.New(uuid, "BedrockPlayer", player.ClientEditionBedrock)
	if err := applyPersistedPlayerData(restored, saved); err != nil {
		t.Fatalf("apply: %v", err)
	}

	if restored.Position != source.Position || restored.Rotation != source.Rotation || restored.Dimension != source.Dimension {
		t.Fatalf("location mismatch: got %+v/%+v dim %d, want %+v/%+v dim %d", restored.Position, restored.Rotation, restored.Dimension, source.Position, source.Rotation, source.Dimension)
	}
	if !reflect.DeepEqual(restored.Inventory, source.Inventory) {
		t.Fatalf("inventory mismatch: got %#v, want %#v", restored.Inventory, source.Inventory)
	}
	if !reflect.DeepEqual(restored.EnderChestInventory, source.EnderChestInventory) {
		t.Fatalf("ender chest mismatch: got %#v, want %#v", restored.EnderChestInventory, source.EnderChestInventory)
	}
	if restored.HeldSlot != source.HeldSlot || restored.GameMode != source.GameMode {
		t.Fatalf("player settings mismatch: slot/mode %d/%d, want %d/%d", restored.HeldSlot, restored.GameMode, source.HeldSlot, source.GameMode)
	}
	health, food, saturation, _ := restored.HealthSnapshot()
	_, _, exhaustion := restored.HungerSnapshot()
	if health != source.Health || food != source.Food || saturation != source.Saturation || exhaustion != source.Exhaustion {
		t.Fatalf("survival state mismatch: got %v/%v/%v/%v", health, food, saturation, exhaustion)
	}
	if !restored.HasSpawnPoint || restored.SpawnPoint != source.SpawnPoint {
		t.Fatalf("spawn point mismatch: got %+v/%t", restored.SpawnPoint, restored.HasSpawnPoint)
	}
}

func TestReconnectFromDeathScreenRespawnsAtFullHealth(t *testing.T) {
	store := newPlayerDataStore(t.TempDir())
	uuid := [16]byte{0xde, 0xad}
	dead := player.New(uuid, "dead-player", player.ClientEditionJava)
	dead.Position = spatial.Vec3{X: 80.5, Y: 12, Z: -20.5}
	dead.Dimension = dimensionEnd
	dead.ApplyDamage(dead.MaxHealth, "test")
	if err := store.save(uuid, snapshotPlayerData(dead)); err != nil {
		t.Fatal(err)
	}

	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	restored := player.New(uuid, "dead-player", player.ClientEditionJava)
	restored.WorldSpawn = spatial.Vec3{X: 0.5, Y: 65, Z: 0.5}
	s := &Server{world: w, playerStore: store}
	s.loadPlayerData(restored)
	health, food, saturation, isDead := restored.HealthSnapshot()
	if isDead || health != restored.MaxHealth || food != 20 || saturation != 5 {
		t.Fatalf("restored survival state = health %.1f food %d saturation %.1f dead %t", health, food, saturation, isDead)
	}
	if restored.Dimension != dimensionOverworld || restored.Position != restored.WorldSpawn {
		t.Fatalf("restored location = dim %d pos %+v, want overworld spawn %+v", restored.Dimension, restored.Position, restored.WorldSpawn)
	}
}

func TestLegacyZeroHealthReconnectRespawnsAtFullHealth(t *testing.T) {
	store := newPlayerDataStore(t.TempDir())
	uuid := [16]byte{0xfa, 0xce}
	legacy := snapshotPlayerData(player.New(uuid, "legacy-dead", player.ClientEditionJava))
	legacy.Health = 0
	legacy.Dead = false
	if err := store.save(uuid, legacy); err != nil {
		t.Fatal(err)
	}

	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	restored := player.New(uuid, "legacy-dead", player.ClientEditionJava)
	restored.WorldSpawn = spatial.Vec3{X: 0.5, Y: 65, Z: 0.5}
	s := &Server{world: w, playerStore: store}
	s.loadPlayerData(restored)

	health, food, saturation, dead := restored.HealthSnapshot()
	if dead || health != 20 || food != 20 || saturation != 5 {
		t.Fatalf("legacy death restored as health %.1f food %d saturation %.1f dead %t", health, food, saturation, dead)
	}
}

func TestPlayerDataStoreMissingPlayer(t *testing.T) {
	store := newPlayerDataStore(t.TempDir())
	_, found, err := store.load([16]byte{1})
	if err != nil || found {
		t.Fatalf("missing load = found %t, err %v", found, err)
	}
}
