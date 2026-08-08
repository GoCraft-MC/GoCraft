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

func TestBedrockBreakingDoublePlantRemovesBothHalves(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{11}, "bedrock-gardener", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	lower := coreworld.Block{Namespace: "minecraft", Name: "peony", Properties: map[string]string{"half": "lower"}}
	upper := coreworld.Block{Namespace: "minecraft", Name: "peony", Properties: map[string]string{"half": "upper"}}
	w.SetBlock(0, 64, 0, lower)
	w.SetBlock(0, 65, 0, upper)
	s := &Server{game: g, world: w, sessions: session.NewManager()}

	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID,
		Action:     intent.BlockActionBreak,
		Position:   spatial.BlockPos{X: 0, Y: 64, Z: 0},
	})

	for y := 64; y <= 65; y++ {
		if got := w.GetBlock(0, y, 0); !got.IsAir() {
			t.Fatalf("plant half y=%d = %q, want air", y, got.ResourceLocation())
		}
	}
}

func TestBedrockPlayerStateAcceptsSurvivalSprintButRejectsFlight(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{12}, `bedrock-runner`, player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}

	s.applyBedrockPlayerState(intent.PlayerStateIntent{
		PlayerUUID: p.UUID,
		State:      intent.PlayerStateSprinting,
		Enabled:    true,
	})
	s.applyBedrockPlayerState(intent.PlayerStateIntent{
		PlayerUUID: p.UUID,
		State:      intent.PlayerStateFlying,
		Enabled:    true,
	})
	s.applyBedrockPlayerState(intent.PlayerStateIntent{
		PlayerUUID: p.UUID,
		State:      intent.PlayerStateSneaking,
		Enabled:    true,
	})
	if !p.Sprinting {
		t.Fatal(`Bedrock sprint transition was not accepted`)
	}
	if p.Flying {
		t.Fatal(`survival Bedrock player was allowed to fly`)
	}
	if !p.Sneaking {
		t.Fatal(`Bedrock sneak transition was not accepted`)
	}
}

func TestBedrockPlayerStateAcceptsCreativeFlight(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{13}, `bedrock-flyer`, player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}

	s.applyBedrockPlayerState(intent.PlayerStateIntent{
		PlayerUUID: p.UUID,
		State:      intent.PlayerStateFlying,
		Enabled:    true,
	})
	if !p.Flying {
		t.Fatal(`creative Bedrock flight transition was rejected`)
	}
}

func TestBedrockBreakingLogAwardsLog(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{14}, "bedrock-lumberjack", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "oak_log", Properties: map[string]string{"axis": "y"}})
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID,
		Action:     intent.BlockActionBreak,
		Position:   spatial.BlockPos{X: 1, Y: 64, Z: 0},
	})
	if got := w.GetBlock(1, 64, 0); !got.IsAir() {
		t.Fatalf("log remained as %q", got.ResourceLocation())
	}
	for _, stack := range p.Inventory {
		if stack.ItemID == "minecraft:oak_log" && stack.Count == 1 {
			return
		}
	}
	t.Fatal("broken oak log was not awarded to Bedrock inventory")
}

func TestBedrockStoneUsesVanillaLootAndHarvestTool(t *testing.T) {
	for _, test := range []struct {
		name     string
		tool     string
		wantDrop bool
	}{
		{name: "hand", wantDrop: false},
		{name: "pickaxe", tool: "minecraft:wooden_pickaxe", wantDrop: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
			defer w.Close()
			g := game.New()
			p := player.New([16]byte{24}, "bedrock-miner", player.ClientEditionBedrock)
			p.GameMode = player.GameModeSurvival
			p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
			if test.tool != "" {
				p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: test.tool, Count: 1}
			}
			if err := g.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
			w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "stone"})
			s := &Server{game: g, world: w, sessions: session.NewManager()}
			s.applyBedrockBlockInteract(intent.BlockInteractIntent{
				PlayerUUID: p.UUID,
				Action:     intent.BlockActionBreak,
				Position:   spatial.BlockPos{X: 1, Y: 64, Z: 0},
			})
			got := false
			for _, stack := range p.Inventory {
				got = got || stack.ItemID == "minecraft:cobblestone" && stack.Count == 1
			}
			if got != test.wantDrop {
				t.Fatalf("cobblestone present = %v, want %v", got, test.wantDrop)
			}
		})
	}
}

func TestBedrockBreakPreservesSelectedToolAndDamagesIt(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{16}, "bedrock-digger", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	p.HeldSlot = 4
	p.Inventory[player.HotbarStart+4] = player.ItemStack{ItemID: "minecraft:wooden_shovel", Count: 1}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})
	s := &Server{game: g, world: w, sessions: session.NewManager()}

	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID,
		Action:     intent.BlockActionBreak,
		Position:   spatial.BlockPos{X: 1, Y: 64, Z: 0},
		HotbarSlot: -1,
	})

	if p.HeldSlot != 4 {
		t.Fatalf("selected slot after break = %d, want 4", p.HeldSlot)
	}
	if got := p.Inventory[player.HotbarStart+4].Damage; got != 1 {
		t.Fatalf("selected shovel damage = %d, want 1", got)
	}
	if got := p.Inventory[player.HotbarStart].Damage; got != 0 {
		t.Fatalf("slot-zero item damage = %d, want 0", got)
	}
}

func TestBedrockStaleActionSlotDoesNotOverrideCurrentSelection(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{17}, "bedrock-fast-scroller", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	p.HeldSlot = 7
	p.Inventory[player.HotbarStart+2] = player.ItemStack{ItemID: "minecraft:wooden_shovel", Count: 1}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})
	s := &Server{game: g, world: w, sessions: session.NewManager()}

	// The action was performed with slot 2, but a subsequent MobEquipment has
	// already selected slot 7 by the time the simulation applies the action.
	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID,
		Action:     intent.BlockActionBreak,
		Position:   spatial.BlockPos{X: 1, Y: 64, Z: 0},
		HotbarSlot: 2,
	})

	if p.HeldSlot != 7 {
		t.Fatalf("selected slot after stale action = %d, want 7", p.HeldSlot)
	}
	if got := p.Inventory[player.HotbarStart+2].Damage; got != 1 {
		t.Fatalf("action shovel damage = %d, want 1", got)
	}
}

func TestBedrockCanConsumeFood(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{15}, "bedrock-eater", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Food = 14
	p.Saturation = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:bread", Count: 2}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}
	s.applyBedrockConsumeFood(intent.ConsumeFoodIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	_, food, saturation, _ := p.HealthSnapshot()
	if food != 19 || saturation != 6 {
		t.Fatalf("after eating bread = food %d saturation %.1f, want 19/6", food, saturation)
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("bread count = %d, want 1", got)
	}
}
