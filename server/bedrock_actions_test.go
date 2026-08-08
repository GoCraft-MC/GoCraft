package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

func newBedrockActionTestServer(t *testing.T) (*Server, *player.Player) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { w.Close() })
	g := game.New()
	p := player.New([16]byte{31}, "bedrock-builder", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	return &Server{game: g, world: w, sessions: session.NewManager()}, p
}

func TestEveryHoeTillsBedrockDirtAndFarmlandHydrates(t *testing.T) {
	hoes := []string{"wooden_hoe", "stone_hoe", "iron_hoe", "golden_hoe", "diamond_hoe", "netherite_hoe"}
	for index, hoe := range hoes {
		t.Run(hoe, func(t *testing.T) {
			s, p := newBedrockActionTestServer(t)
			x := index + 1
			s.world.SetBlock(x, 64, 0, bedrockBlock("dirt", nil))
			s.world.SetBlock(x+4, 64, 0, coreworld.MakeFluid("minecraft:water", 0))
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:" + hoe, Count: 1}

			used := s.applyBedrockItemAction(p, intent.BlockInteractIntent{
				Position: spatial.BlockPos{X: int32(x), Y: 64, Z: 0}, Face: 1,
			}, s.world.GetBlock(x, 64, 0))
			if !used {
				t.Fatal("hoe click was not handled")
			}
			farmland := s.world.GetBlock(x, 64, 0)
			if farmland.ResourceLocation() != "minecraft:farmland" || farmland.Properties["moisture"] != "0" {
				t.Fatalf("tilled block = %+v, want dry farmland", farmland)
			}
			if got := p.Inventory[player.HotbarStart].Damage; got != 1 {
				t.Fatalf("hoe damage = %d, want 1", got)
			}
			for tick := int64(20); tick <= 400 && s.world.GetBlock(x, 64, 0).Properties["moisture"] != "7"; tick += 20 {
				s.world.TickFarmland(tick, 64)
			}
			if got := s.world.GetBlock(x, 64, 0).Properties["moisture"]; got != "7" {
				t.Fatalf("farmland moisture = %q, want 7 with nearby water", got)
			}
		})
	}
}

func TestBedrockRootedDirtHoeDropsRoots(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	s.world.SetBlock(1, 64, 0, bedrockBlock("rooted_dirt", nil))
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:wooden_hoe", Count: 1}
	if !s.applyBedrockItemAction(p, intent.BlockInteractIntent{
		Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}, Face: 0,
	}, s.world.GetBlock(1, 64, 0)) {
		t.Fatal("rooted dirt was not handled from its underside")
	}
	if got := s.world.GetBlock(1, 64, 0).ResourceLocation(); got != "minecraft:dirt" {
		t.Fatalf("rooted dirt became %q, want dirt", got)
	}
	for _, stack := range p.Inventory {
		if stack.ItemID == "minecraft:hanging_roots" && stack.Count == 1 {
			return
		}
	}
	t.Fatal("hanging roots were not awarded")
}

func TestBedrockTorchChoosesFloorAndWallStates(t *testing.T) {
	t.Run("floor", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 63, 0, bedrockBlock("stone", nil))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:torch", Count: 2}
		if !s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
			Position: spatial.BlockPos{X: 1, Y: 63, Z: 0}, Face: 1,
		}, s.world.GetBlock(1, 63, 0)) {
			t.Fatal("floor torch click was not handled")
		}
		if got := s.world.GetBlock(1, 64, 0).ResourceLocation(); got != "minecraft:torch" {
			t.Fatalf("placed %q, want floor torch", got)
		}
		if got := p.Inventory[player.HotbarStart].Count; got != 1 {
			t.Fatalf("torch count = %d, want 1", got)
		}
	})

	t.Run("wall", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 64, 0, bedrockBlock("stone", nil))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:torch", Count: 1}
		s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
			Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}, Face: 2,
		}, s.world.GetBlock(1, 64, 0))
		placed := s.world.GetBlock(1, 64, -1)
		if placed.ResourceLocation() != "minecraft:wall_torch" || placed.Properties["facing"] != "north" {
			t.Fatalf("wall torch = %+v, want north-facing wall torch", placed)
		}
	})
}

func TestBedrockMechanismsToggleAndButtonReleases(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	s.world.SetBlock(1, 64, 0, bedrockBlock("lever", map[string]string{"face": "wall", "facing": "north", "powered": "false"}))
	if !s.applyBedrockBlockActivation(p, spatial.BlockPos{X: 1, Y: 64, Z: 0}, s.world.GetBlock(1, 64, 0)) {
		t.Fatal("lever activation was not handled")
	}
	if got := s.world.GetBlock(1, 64, 0).Properties["powered"]; got != "true" {
		t.Fatalf("lever powered = %q, want true", got)
	}

	s.world.SetBlock(2, 64, 0, bedrockBlock("stone_button", map[string]string{"face": "wall", "facing": "north", "powered": "false"}))
	s.applyBedrockBlockActivation(p, spatial.BlockPos{X: 2, Y: 64, Z: 0}, s.world.GetBlock(2, 64, 0))
	if got := s.world.GetBlock(2, 64, 0).Properties["powered"]; got != "true" {
		t.Fatalf("button powered = %q, want true", got)
	}
	s.worldAge = 20
	s.tickBlockPhysics()
	if got := s.world.GetBlock(2, 64, 0).Properties["powered"]; got != "false" {
		t.Fatalf("button powered after 20 ticks = %q, want false", got)
	}

	s.world.SetBlock(3, 64, 0, bedrockBlock("oak_trapdoor", map[string]string{"facing": "north", "half": "bottom", "open": "false", "powered": "false"}))
	s.applyBedrockBlockActivation(p, spatial.BlockPos{X: 3, Y: 64, Z: 0}, s.world.GetBlock(3, 64, 0))
	if got := s.world.GetBlock(3, 64, 0).Properties["open"]; got != "true" {
		t.Fatalf("trapdoor open = %q, want true", got)
	}
}

func TestBedrockPlacesDoorBedAndRedstoneDust(t *testing.T) {
	t.Run("door", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 63, 0, bedrockBlock("stone", nil))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_door", Count: 1}
		s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
			Position: spatial.BlockPos{X: 1, Y: 63, Z: 0}, Face: 1, ClickX: 0.25, ClickZ: 0.5,
		}, s.world.GetBlock(1, 63, 0))
		lower, upper := s.world.GetBlock(1, 64, 0), s.world.GetBlock(1, 65, 0)
		if lower.ResourceLocation() != "minecraft:oak_door" || lower.Properties["half"] != "lower" || upper.Properties["half"] != "upper" {
			t.Fatalf("door halves = %+v / %+v", lower, upper)
		}
	})

	t.Run("bed", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 63, 0, bedrockBlock("stone", nil))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:white_bed", Count: 1}
		s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
			Position: spatial.BlockPos{X: 1, Y: 63, Z: 0}, Face: 1,
		}, s.world.GetBlock(1, 63, 0))
		foot := s.world.GetBlock(1, 64, 0)
		dx, dz := bedrockHorizontalOffset(foot.Properties["facing"])
		head := s.world.GetBlock(1+dx, 64, dz)
		if foot.Properties["part"] != "foot" || head.ResourceLocation() != "minecraft:white_bed" || head.Properties["part"] != "head" {
			t.Fatalf("bed halves = %+v / %+v", foot, head)
		}
	})

	t.Run("redstone", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 63, 0, bedrockBlock("stone", nil))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:redstone", Count: 1}
		s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
			Position: spatial.BlockPos{X: 1, Y: 63, Z: 0}, Face: 1,
		}, s.world.GetBlock(1, 63, 0))
		wire := s.world.GetBlock(1, 64, 0)
		if wire.ResourceLocation() != "minecraft:redstone_wire" || wire.Properties["power"] != "0" {
			t.Fatalf("redstone placement = %+v", wire)
		}
	})
}

func TestBedrockLightBlockKeepsSelectedLevel(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	s.world.SetBlock(1, 63, 0, bedrockBlock("stone", nil))
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:light", Count: 1, Damage: 11}
	if !s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
		Position: spatial.BlockPos{X: 1, Y: 63, Z: 0}, Face: 1,
	}, s.world.GetBlock(1, 63, 0)) {
		t.Fatal("light block click was not handled")
	}
	light := s.world.GetBlock(1, 64, 0)
	if light.ResourceLocation() != "minecraft:light" || light.Properties["level"] != "11" {
		t.Fatalf("placed light = %+v, want level 11", light)
	}
}

func TestBedrockAttachedBlocksRequireSupport(t *testing.T) {
	for _, item := range []string{"minecraft:torch", "minecraft:lever", "minecraft:stone_button", "minecraft:ladder"} {
		t.Run(item, func(t *testing.T) {
			s, p := newBedrockActionTestServer(t)
			p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: item, Count: 1}
			s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
				Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}, Face: 2,
			}, s.world.GetBlock(1, 64, 0))
			if got := s.world.GetBlock(1, 64, 0); !got.IsAir() {
				t.Fatalf("unsupported item placed as %+v", got)
			}
			if got := p.Inventory[player.HotbarStart].Count; got != 1 {
				t.Fatalf("unsupported placement consumed item; count=%d", got)
			}
		})
	}
}

func TestSneakingPlacesHeldBlockInsteadOfActivatingMechanism(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	p.Sneaking = true
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:stone", Count: 1}
	s.world.SetBlock(1, 64, 0, bedrockBlock("lever", map[string]string{"face": "floor", "facing": "north", "powered": "false"}))
	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID, Action: intent.BlockActionUse,
		Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}, Face: 1, HotbarSlot: 0,
	})
	if got := s.world.GetBlock(1, 64, 0).Properties["powered"]; got != "false" {
		t.Fatalf("lever powered = %q, want false while bypassing activation", got)
	}
	if got := s.world.GetBlock(1, 65, 0).ResourceLocation(); got != "minecraft:stone" {
		t.Fatalf("sneak placement = %q, want minecraft:stone", got)
	}
}

func TestBedrockRedstoneDustConnectsToNeighbour(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	for x := 1; x <= 2; x++ {
		s.world.SetBlock(x, 63, 0, bedrockBlock("stone", nil))
	}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:redstone", Count: 2}
	for x := 1; x <= 2; x++ {
		s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
			Position: spatial.BlockPos{X: int32(x), Y: 63, Z: 0}, Face: 1,
		}, s.world.GetBlock(x, 63, 0))
	}
	left, right := s.world.GetBlock(1, 64, 0), s.world.GetBlock(2, 64, 0)
	if left.Properties["east"] != "side" || right.Properties["west"] != "side" {
		t.Fatalf("wire connections = left %v right %v", left.Properties, right.Properties)
	}
}

func TestBedrockPumpkinShearsAndComposterLifecycle(t *testing.T) {
	t.Run("pumpkin", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 64, 0, bedrockBlock("pumpkin", nil))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:shears", Count: 1}
		if !s.applyBedrockItemAction(p, intent.BlockInteractIntent{Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}}, s.world.GetBlock(1, 64, 0)) {
			t.Fatal("shears did not carve pumpkin")
		}
		if got := s.world.GetBlock(1, 64, 0).ResourceLocation(); got != "minecraft:carved_pumpkin" {
			t.Fatalf("pumpkin became %q", got)
		}
		var seeds int
		for _, stack := range p.Inventory {
			if stack.ItemID == "minecraft:pumpkin_seeds" {
				seeds += stack.Count
			}
		}
		if seeds != 4 {
			t.Fatalf("pumpkin seeds = %d, want 4", seeds)
		}
	})

	t.Run("composter", func(t *testing.T) {
		s, p := newBedrockActionTestServer(t)
		s.world.SetBlock(1, 64, 0, bedrockBlock("composter", map[string]string{"level": "0"}))
		p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:wheat_seeds", Count: 1}
		s.applyBedrockItemAction(p, intent.BlockInteractIntent{Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}}, s.world.GetBlock(1, 64, 0))
		if got := s.world.GetBlock(1, 64, 0).Properties["level"]; got != "1" {
			t.Fatalf("first compost level = %q, want 1", got)
		}

		s.world.SetBlock(1, 64, 0, bedrockBlock("composter", map[string]string{"level": "7"}))
		s.world.BlockPhysics.ScheduleComposter(1, 64, 0, 0, 20)
		s.worldAge = 20
		s.tickBlockPhysics()
		if got := s.world.GetBlock(1, 64, 0).Properties["level"]; got != "8" {
			t.Fatalf("ready compost level = %q, want 8", got)
		}
		s.applyBedrockBlockActivation(p, spatial.BlockPos{X: 1, Y: 64, Z: 0}, s.world.GetBlock(1, 64, 0))
		if got := s.world.GetBlock(1, 64, 0).Properties["level"]; got != "0" {
			t.Fatalf("emptied compost level = %q, want 0", got)
		}
	})
}

func TestBedrockBucketFillsAndEmptiesCauldron(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	s.world.SetBlock(1, 64, 0, bedrockBlock("cauldron", nil))
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:water_bucket", Count: 1}
	s.applyBedrockItemAction(p, intent.BlockInteractIntent{Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}}, s.world.GetBlock(1, 64, 0))
	filled := s.world.GetBlock(1, 64, 0)
	if filled.ResourceLocation() != "minecraft:water_cauldron" || filled.Properties["level"] != "3" {
		t.Fatalf("filled cauldron = %+v", filled)
	}
	if got := p.Inventory[player.HotbarStart].ItemID; got != "minecraft:bucket" {
		t.Fatalf("held item after filling = %q, want bucket", got)
	}
	s.applyBedrockItemAction(p, intent.BlockInteractIntent{Position: spatial.BlockPos{X: 1, Y: 64, Z: 0}}, filled)
	if got := s.world.GetBlock(1, 64, 0).ResourceLocation(); got != "minecraft:cauldron" {
		t.Fatalf("emptied cauldron = %q", got)
	}
	if got := p.Inventory[player.HotbarStart].ItemID; got != "minecraft:water_bucket" {
		t.Fatalf("held item after emptying = %q, want water bucket", got)
	}
}

func TestBedrockPressurePlateTracksPlayerOccupancy(t *testing.T) {
	s, p := newBedrockActionTestServer(t)
	s.world.SetBlock(1, 63, 0, bedrockBlock("stone", nil))
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_pressure_plate", Count: 1}
	s.placeBedrockHeldBlock(p, intent.BlockInteractIntent{
		Position: spatial.BlockPos{X: 1, Y: 63, Z: 0}, Face: 1,
	}, s.world.GetBlock(1, 63, 0))
	p.Position = spatial.Vec3{X: 1.5, Y: 65, Z: 0.5}
	s.worldAge = 1
	s.tickBlockPhysics()
	if got := s.world.GetBlock(1, 64, 0).Properties["powered"]; got != "true" {
		t.Fatalf("occupied pressure plate powered = %q, want true", got)
	}
	p.Position = spatial.Vec3{X: 4, Y: 65, Z: 4}
	s.worldAge = 3
	s.tickBlockPhysics()
	if got := s.world.GetBlock(1, 64, 0).Properties["powered"]; got != "false" {
		t.Fatalf("empty pressure plate powered = %q, want false", got)
	}
}
