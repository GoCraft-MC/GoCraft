package server

import (
	"testing"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func TestDaylightBurnsExposedUndeadButNotShelteredMob(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	s := &Server{world: world, worldAge: 6000, sessions: session.NewManager()}

	exposed := corentity.New(1, [16]byte{}, corentity.TypeZombie, 0.5, 64, 0.5)
	var hurt []*corentity.Entity
	for tick := 0; tick < 500 && !exposed.Dead; tick++ {
		s.tickMobSunlight(exposed, &hurt)
	}
	if !exposed.Dead || len(hurt) == 0 {
		t.Fatalf("exposed zombie did not burn to death: health=%.1f fire=%d hurt=%d", exposed.Health, exposed.FireTicks, len(hurt))
	}

	sheltered := corentity.New(2, [16]byte{}, corentity.TypeSkeleton, 2.5, 64, 0.5)
	world.SetBlock(2, 66, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	for tick := 0; tick < 200; tick++ {
		s.tickMobSunlight(sheltered, &hurt)
	}
	if sheltered.FireTicks != 0 || sheltered.Health != sheltered.MaxHealth {
		t.Fatalf("sheltered skeleton burned: health=%.1f fire=%d", sheltered.Health, sheltered.FireTicks)
	}
}

func TestDaylightBurnTagMatchesPumpkinUndeadVariants(t *testing.T) {
	for _, entityType := range []corentity.EntityType{
		corentity.TypeSkeleton, corentity.TypeStray, corentity.TypeWitherSkeleton,
		corentity.TypeBogged, corentity.TypeZombie, corentity.TypeZombieHorse,
		corentity.TypeZombieVillager, corentity.TypeDrowned, corentity.TypePhantom,
	} {
		if !burnsInDaylight(entityType) {
			t.Errorf("%s missing daylight-burn behaviour", entityType)
		}
	}
	if burnsInDaylight(corentity.TypeHusk) {
		t.Fatal("husk should be immune to daylight burning")
	}
}

func TestSolidWallPreventsHostileMeleeDamage(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	g := game.New()
	target := player.New([16]byte{1}, "bedrock", player.ClientEditionBedrock)
	target.GameMode = player.GameModeSurvival
	target.Position = spatial.Vec3{X: 2.2, Y: 64, Z: 0.5}
	if err := g.AddPlayer(target); err != nil {
		t.Fatal(err)
	}
	zombie := corentity.New(g.NextEntityID(), [16]byte{2}, corentity.TypeZombie, 0.5, 64, 0.5)
	zombie.OnGround = true
	world.Entities.Add(zombie)
	world.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	world.SetBlock(1, 65, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
	s := &Server{
		cfg: &config.Config{Difficulty: "normal"}, game: g, world: world,
		sessions: session.NewManager(), mobAIs: make(map[int32]*mobAI),
	}

	s.tickHostileMobAI(zombie)
	if target.Health != target.MaxHealth {
		t.Fatalf("zombie damaged player through wall: health=%.1f", target.Health)
	}
	world.SetBlock(1, 64, 0, coreworld.Air)
	world.SetBlock(1, 65, 0, coreworld.Air)
	s.tickHostileMobAI(zombie)
	if target.Health >= target.MaxHealth {
		t.Fatal("zombie did not damage visible player in melee range")
	}
}

func TestSkeletonDrawsBowAndSpawnsArrow(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	g := game.New()
	target := player.New([16]byte{1}, "target", player.ClientEditionBedrock)
	target.GameMode = player.GameModeSurvival
	target.Position = spatial.Vec3{X: 10.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(target); err != nil {
		t.Fatal(err)
	}
	skeleton := corentity.New(g.NextEntityID(), [16]byte{2}, corentity.TypeSkeleton, 0.5, 64, 0.5)
	world.Entities.Add(skeleton)
	s := &Server{
		cfg: &config.Config{Difficulty: "normal"}, game: g, world: world,
		sessions: session.NewManager(), mobAIs: make(map[int32]*mobAI),
	}

	s.tickHostileMobAI(skeleton)
	if skeleton.MainHandItemID != "minecraft:bow" || !skeleton.UsingItem {
		t.Fatalf("skeleton did not begin drawing its equipped bow: %+v", skeleton)
	}
	for tick := 0; tick < 20; tick++ {
		s.tickHostileMobAI(skeleton)
	}
	if skeleton.UsingItem {
		t.Fatal("skeleton remained in bow-use animation after firing")
	}
	for _, entity := range world.Entities.Snapshot() {
		if entity.Type == corentity.TypeArrow && entity.OwnerEntityID == skeleton.EntityID {
			return
		}
	}
	t.Fatal("skeleton bow attack spawned no owned arrow")
}

func TestHostileAIIgnoresPlayersInOtherDimensions(t *testing.T) {
	world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer world.Close()
	world.Chunk(0, 0)
	g := game.New()
	overworldPlayer := player.New([16]byte{31}, "overworld", player.ClientEditionBedrock)
	overworldPlayer.GameMode = player.GameModeSurvival
	overworldPlayer.Dimension = dimensionOverworld
	overworldPlayer.Position = spatial.Vec3{X: 1.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(overworldPlayer); err != nil {
		t.Fatal(err)
	}
	zombie := corentity.New(g.NextEntityID(), [16]byte{32}, corentity.TypeZombie, 0.5, 64, 0.5)
	zombie.OnGround = true
	world.Entities.Add(zombie)
	s := &Server{
		cfg: &config.Config{Difficulty: "normal"}, game: g, world: world,
		sessions: session.NewManager(), mobAIs: make(map[int32]*mobAI), simulationDimension: dimensionNether,
	}

	s.tickHostileMobAI(zombie)
	if overworldPlayer.Health != overworldPlayer.MaxHealth {
		t.Fatalf("Nether zombie damaged Overworld player: health=%.1f", overworldPlayer.Health)
	}
	if got := s.closestVisiblePlayer(zombie, 16); got != nil {
		t.Fatalf("Nether visibility selected player from dimension %d", got.Dimension)
	}
	if got := s.naturalSpawnPlayers(); len(got) != 0 {
		t.Fatalf("Nether spawn set included %d Overworld player(s)", len(got))
	}
}

func TestRangedHostileFamiliesSpawnTheirProjectile(t *testing.T) {
	tests := []struct {
		name       string
		mob        corentity.EntityType
		projectile corentity.EntityType
	}{
		{"blaze", corentity.TypeBlaze, corentity.TypeSmallFireball},
		{"ghast", corentity.TypeGhast, corentity.TypeFireball},
		{"breeze", corentity.TypeBreeze, corentity.TypeWindCharge},
		{"witch", corentity.TypeWitch, corentity.TypePotion},
		{"pillager", corentity.TypePillager, corentity.TypeArrow},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			world := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer world.Close()
			world.Chunk(0, 0)
			g := game.New()
			target := player.New([16]byte{byte(index + 70)}, "target", player.ClientEditionBedrock)
			target.GameMode = player.GameModeSurvival
			target.Position = spatial.Vec3{X: 8.5, Y: 64, Z: 0.5}
			if err := g.AddPlayer(target); err != nil {
				t.Fatal(err)
			}
			mob := corentity.New(g.NextEntityID(), [16]byte{byte(index + 80)}, test.mob, 0.5, 64, 0.5)
			world.Entities.Add(mob)
			s := &Server{
				cfg: &config.Config{Difficulty: "normal"}, game: g, world: world,
				sessions: session.NewManager(), mobAIs: make(map[int32]*mobAI),
			}

			s.tickHostileMobAI(mob)
			for _, spawned := range world.Entities.Snapshot() {
				if spawned.Type == test.projectile && spawned.OwnerEntityID == mob.EntityID {
					return
				}
			}
			t.Fatalf("%s spawned no %s", test.mob, test.projectile)
		})
	}
}
