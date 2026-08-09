package server

import (
	"math"
	"math/rand"
	"testing"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestNaturalSpawnerProducesLandMobsInLoadedSimulationChunks(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	for cx := int32(-8); cx <= 8; cx++ {
		for cz := int32(-8); cz <= 8; cz++ {
			world.Chunk(cx, cz)
		}
	}
	gameCore := game.New()
	p := player.New([16]byte{1}, "spawner", player.ClientEditionJava)
	p.EntityID = 1
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := gameCore.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{
		cfg:      &config.Config{ViewDistance: 8, Difficulty: "normal"},
		game:     gameCore,
		world:    world,
		sessions: session.NewManager(),
		spawnRNG: rand.New(rand.NewSource(1)),
	}
	for tick := int64(1); tick <= 800; tick++ {
		s.worldAge = tick + 18000
		s.tickNaturalSpawning()
	}
	var hostile, passive int
	for _, spawned := range world.Entities.Snapshot() {
		if isHostileMob(spawned.Type) {
			hostile++
		}
		if settings, ok := pumpkinEntitySpawnSettingsByType[string(spawned.Type)]; ok && settings.category == mobCategoryCreature {
			passive++
		}
	}
	if hostile == 0 || passive == 0 {
		t.Fatalf("natural spawner counts after 800 ticks: hostile=%d passive=%d", hostile, passive)
	}
}

func TestNaturalSpawnerProducesMobsInGeneratedOverworld(t *testing.T) {
	world := coreworld.New(coreworld.NewOverworldGenerator(0), nil, false)
	defer world.Close()
	for cx := int32(-4); cx <= 4; cx++ {
		for cz := int32(-4); cz <= 4; cz++ {
			world.Chunk(cx, cz)
		}
	}
	gameCore := game.New()
	p := player.New([16]byte{1}, "spawner", player.ClientEditionJava)
	p.EntityID = 1
	p.Position = spatial.Vec3{X: 0.5, Y: 80, Z: 0.5}
	if err := gameCore.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{
		cfg:      &config.Config{ViewDistance: 4, Difficulty: "normal"},
		game:     gameCore,
		world:    world,
		sessions: session.NewManager(),
		spawnRNG: rand.New(rand.NewSource(1)),
	}
	for tick := int64(1); tick <= 800; tick++ {
		s.worldAge = tick + 18000
		s.tickNaturalSpawning()
	}
	var hostile, passive int
	for _, spawned := range world.Entities.Snapshot() {
		if isHostileMob(spawned.Type) {
			hostile++
		}
		if settings, ok := pumpkinEntitySpawnSettingsByType[string(spawned.Type)]; ok && settings.category == mobCategoryCreature {
			passive++
			x, z := int(math.Floor(spawned.Position.X)), int(math.Floor(spawned.Position.Z))
			surface := world.SurfaceY(x, z)
			if got, want := int(spawned.Position.Y), pumpkinSurfaceSpawnY(world, x, z, surface); got != want {
				t.Errorf("passive %s spawned underground at Y=%d, surface spawn Y=%d", spawned.Type, got, want)
			}
		}
	}
	if hostile == 0 || passive == 0 {
		t.Fatalf("generated-overworld natural spawn counts: hostile=%d passive=%d", hostile, passive)
	}
}

func TestNaturalSpawnerProducesFishInGeneratedOcean(t *testing.T) {
	generator := coreworld.NewOverworldGenerator(0)
	oceanX, oceanZ, ok := generator.NearestBiome(0, 0, "minecraft:ocean", 8192)
	if !ok {
		t.Fatal("seed 0 has no ocean in lookup radius")
	}
	world := coreworld.New(generator, nil, false)
	defer world.Close()
	centerCX, centerCZ := coreworld.ChunkCoordsFor(oceanX, oceanZ)
	for dx := int32(-4); dx <= 4; dx++ {
		for dz := int32(-4); dz <= 4; dz++ {
			world.Chunk(centerCX+dx, centerCZ+dz)
		}
	}
	gameCore := game.New()
	p := player.New([16]byte{1}, "swimmer", player.ClientEditionJava)
	p.EntityID = 1
	p.Position = spatial.Vec3{X: float64(oceanX) + 0.5, Y: float64(coreworld.SeaLevel + 1), Z: float64(oceanZ) + 0.5}
	if err := gameCore.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{
		cfg:      &config.Config{ViewDistance: 4, Difficulty: "normal"},
		game:     gameCore,
		world:    world,
		sessions: session.NewManager(),
		spawnRNG: rand.New(rand.NewSource(2)),
	}
	for tick := int64(1); tick <= 400; tick++ {
		s.worldAge = tick
		s.tickNaturalSpawning()
	}
	for _, spawned := range world.Entities.Snapshot() {
		if isAquaticMob(spawned.Type) {
			return
		}
	}
	t.Fatal("natural spawner produced no aquatic mobs in an ocean")
}

func TestPumpkinCategoryCapsScaleAtSeventeenBySeventeenChunks(t *testing.T) {
	state := naturalSpawnState{spawnableChunkCount: pumpkinSpawnableChunkDenominator}
	want := []int{70, 10, 15, 5, 5, 5, 20, -1}
	for category, expected := range want {
		if got := state.globalCap(mobCategory(category)); got != expected {
			t.Fatalf("category %d cap = %d, want %d", category, got, expected)
		}
	}
}

func TestPumpkinPersistentSpawnCategoriesUseFourHundredTickGate(t *testing.T) {
	state := naturalSpawnState{spawnableChunkCount: pumpkinSpawnableChunkDenominator}
	withoutPassives := filteredSpawningCategories(&state, true, false)
	if containsMobCategory(withoutPassives, mobCategoryCreature) || containsMobCategory(withoutPassives, mobCategoryWaterCreature) {
		t.Fatalf("persistent categories enabled outside passive cycle: %v", withoutPassives)
	}
	withPassives := filteredSpawningCategories(&state, true, true)
	if !containsMobCategory(withPassives, mobCategoryCreature) || !containsMobCategory(withPassives, mobCategoryWaterCreature) {
		t.Fatalf("persistent categories missing on passive cycle: %v", withPassives)
	}
}

func TestGeneratedPumpkinSpawnDataCoversGoCraftBiomesAndEntities(t *testing.T) {
	for _, biome := range coreworld.GeneratedBiomeNames() {
		key := "minecraft:" + biome
		settings, ok := pumpkinBiomeSpawns[key]
		if !ok {
			t.Errorf("missing Pumpkin biome spawn table for %s", key)
			continue
		}
		for _, entries := range settings.groups {
			for _, entry := range entries {
				if _, ok := pumpkinEntitySpawnSettingsByType[entry.entityType]; !ok {
					t.Errorf("%s spawn entry %s has no Pumpkin entity settings", key, entry.entityType)
				}
			}
		}
	}
	for _, entry := range pumpkinBiomeSpawns["minecraft:ocean"].groups[mobCategoryWaterCreature] {
		if entry.entityType == string(corentity.TypeGlowSquid) || entry.entityType == string(corentity.TypeCod) {
			t.Errorf("ocean water-creature table leaked adjacent category entry %s", entry.entityType)
		}
	}
	cow := pumpkinEntitySpawnSettingsByType[string(corentity.TypeCow)]
	if cow.category != mobCategoryCreature || cow.location != spawnLocationOnGround || cow.limitPerChunk != 4 || cow.movementSpeed != 0.20000000298023224 {
		t.Fatalf("generated cow settings = %+v", cow)
	}
	zombie := pumpkinEntitySpawnSettingsByType[string(corentity.TypeZombie)]
	if zombie.followRange != 35 || zombie.attackDamage != 3 {
		t.Fatalf("generated zombie settings = %+v", zombie)
	}
}

func TestDistantNaturalMobsDespawnButGeneratedEntitiesRemain(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	gameCore := game.New()
	p := player.New([16]byte{1}, "observer", player.ClientEditionJava)
	p.EntityID = 1
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := gameCore.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	natural := corentity.New(2, [16]byte{2}, corentity.TypeCow, 200, 64, 0)
	natural.NaturalSpawned = true
	generated := corentity.New(3, [16]byte{3}, corentity.TypeVillager, 200, 64, 0)
	world.Entities.Add(natural)
	world.Entities.Add(generated)
	s := &Server{world: world, game: gameCore, mobAIs: make(map[int32]*mobAI)}
	var removed []int32
	s.despawnDistantNaturalMobs(s.naturalSpawnPlayers(), &removed)
	if _, ok := world.Entities.Get(natural.EntityID); ok {
		t.Error("distant naturally spawned cow was retained")
	}
	if _, ok := world.Entities.Get(generated.EntityID); !ok {
		t.Error("generated villager was despawned")
	}
	if len(removed) != 1 || removed[0] != natural.EntityID {
		t.Fatalf("removed IDs = %v, want [%d]", removed, natural.EntityID)
	}
}

func TestNaturalSpawnPositionHonoursPumpkinPlayerAndWorldSpawnExclusions(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(2, 0)
	s := &Server{world: world, spawnX: 0, spawnZ: 0, worldAge: 6000}
	settings := pumpkinEntitySpawnSettingsByType[string(corentity.TypeCow)]
	farPlayer := []naturalSpawnPlayer{{id: 1, position: spatial.Vec3{X: 100, Y: 64, Z: 0}}}
	if !s.validNaturalSpawnPosition(farPlayer, mobCategoryCreature, string(corentity.TypeCow), settings, 2, 0, 32, 64, 0) {
		t.Fatal("valid loaded grassland position was rejected")
	}
	nearPlayer := []naturalSpawnPlayer{{id: 1, position: spatial.Vec3{X: 40, Y: 64, Z: 0}}}
	if s.validNaturalSpawnPosition(nearPlayer, mobCategoryCreature, string(corentity.TypeCow), settings, 2, 0, 32, 64, 0) {
		t.Fatal("position within 24 blocks of player was accepted")
	}
}

func TestPumpkinTemptGoalUsesNavigator(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	g := game.New()
	p := player.New([16]byte{1}, "farmer", player.ClientEditionJava)
	p.EntityID = 42
	p.Position = spatial.Vec3{X: 8.5, Y: 64, Z: 1.5}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:wheat", Count: 1}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{world: world, game: g, mobAIs: make(map[int32]*mobAI)}
	cow := corentity.New(7, [16]byte{}, corentity.TypeCow, 1.5, 64, 1.5)
	cow.OnGround = true
	ai := s.mobAIFor(cow)
	s.tickPassiveIdleGoals(cow, ai)
	if !ai.hasPathGoal || len(ai.path) == 0 || cow.VX <= 0 {
		t.Fatalf("tempt navigation did not start: ai=%+v velocity=(%.3f, %.3f)", ai, cow.VX, cow.VZ)
	}
}

func TestNewerPumpkinMobsAreSkippedForJava1214Registry(t *testing.T) {
	s := &Server{spawnRNG: rand.New(rand.NewSource(1))}
	entry, ok := s.pumpkinSpawnEntrySupportedByProtocol([]pumpkinSpawnEntry{
		{entityType: "minecraft:nautilus", minCount: 1, maxCount: 1},
		{entityType: string(corentity.TypeSquid), minCount: 1, maxCount: 4},
	})
	if !ok || entry.entityType != string(corentity.TypeSquid) {
		t.Fatalf("protocol-compatible fallback = %+v, present=%v", entry, ok)
	}
	if _, ok := s.pumpkinSpawnEntrySupportedByProtocol([]pumpkinSpawnEntry{{entityType: "minecraft:parched"}}); ok {
		t.Fatal("newer Pumpkin-only entity was accepted by Java 1.21.4 spawner")
	}
}

func containsMobCategory(categories []mobCategory, target mobCategory) bool {
	for _, category := range categories {
		if category == target {
			return true
		}
	}
	return false
}
