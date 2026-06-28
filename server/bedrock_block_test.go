package server

import (
	"testing"
	"time"

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

func TestBedrockBreakingSupportRemovesFloatingGrassImmediately(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{42}, "bedrock-gardener", player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.SetBlock(0, 63, 0, coreworld.Block{Namespace: "minecraft", Name: "dirt"})
	w.SetBlock(0, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "short_grass"})
	s := &Server{game: g, world: w, sessions: session.NewManager()}

	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID, Action: intent.BlockActionBreak,
		Position: spatial.BlockPos{X: 0, Y: 63, Z: 0}, HotbarSlot: 0,
	})
	if support := w.GetBlock(0, 63, 0); !support.IsAir() {
		t.Fatalf("broken support = %q, want air", support.ResourceLocation())
	}
	if grass := w.GetBlock(0, 64, 0); !grass.IsAir() {
		t.Fatalf("grass above broken support = %q, want air", grass.ResourceLocation())
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

func TestBedrockCreativeToolWorksAfterSwitchingToSurvival(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{25}, "bedrock-creative-miner", player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	p.Position = spatial.Vec3{X: 0.5, Y: 64, Z: 0.5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{
		PlayerUUID: p.UUID,
		Actions: []intent.InventoryAction{
			{
				Kind:        intent.InventoryActionCreativeGive,
				Destination: intent.InventoryCursorSlot,
				Count:       1,
				Item:        player.ItemStack{ItemID: "minecraft:iron_pickaxe", Count: 1},
			},
			{
				Kind:        intent.InventoryActionMove,
				Source:      intent.InventoryCursorSlot,
				Destination: player.HotbarStart,
				Count:       1,
			},
		},
		Done: done,
	})
	if result := <-done; !result.Accepted {
		t.Fatal("creative iron pickaxe was rejected")
	}

	// A creative pick becomes an ordinary pristine tool in Survival. Creative
	// mode itself intentionally has neither block drops nor durability wear.
	p.GameMode = player.GameModeSurvival
	w.SetBlock(1, 64, 0, coreworld.Block{Namespace: "minecraft", Name: "iron_ore"})
	s.applyBedrockBlockInteract(intent.BlockInteractIntent{
		PlayerUUID: p.UUID,
		Action:     intent.BlockActionBreak,
		Position:   spatial.BlockPos{X: 1, Y: 64, Z: 0},
		HotbarSlot: 0,
	})

	tool := p.Inventory[player.HotbarStart]
	if tool.ItemID != "minecraft:iron_pickaxe" || tool.Damage != 1 {
		t.Fatalf("creative pickaxe after mining = %+v, want iron pickaxe with one damage", tool)
	}
	for _, stack := range p.Inventory {
		if stack.ItemID == "minecraft:raw_iron" && stack.Count == 1 {
			return
		}
	}
	t.Fatal("creative iron pickaxe did not harvest raw iron after switching to Survival")
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
	s.applyBedrockStartUseItem(intent.StartUseItemIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	if p.UsingItemID != "minecraft:bread" || p.UsingItemSince.IsZero() {
		t.Fatalf("eating did not start: item=%q since=%v", p.UsingItemID, p.UsingItemSince)
	}
	p.UsingItemSince = time.Now().Add(-2 * time.Second)
	s.applyBedrockConsumeFood(intent.ConsumeFoodIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	_, food, saturation, _ := p.HealthSnapshot()
	if food != 19 || saturation != 6 {
		t.Fatalf("after eating bread = food %d saturation %.1f, want 19/6", food, saturation)
	}
	if got := p.Inventory[player.HotbarStart].Count; got != 1 {
		t.Fatalf("bread count = %d, want 1", got)
	}
	if p.UsingItemID != "" || !p.UsingItemSince.IsZero() {
		t.Fatalf("use state remained after eating: %q/%v", p.UsingItemID, p.UsingItemSince)
	}
}

func TestBedrockFoodCompletesOnServerTickAndReturnsContainer(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{43}, "bedrock-hungry", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Food = 10
	p.Saturation = 0
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:honey_bottle", Count: 1}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}
	s.applyBedrockStartUseItem(intent.StartUseItemIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	p.UsingItemSince = time.Now().Add(-2 * time.Second)
	s.tickBedrockItemUse()

	_, food, saturation, _ := p.HealthSnapshot()
	if food != 16 || saturation != 1.2 {
		t.Fatalf("after honey bottle = food %d saturation %.1f, want 16/1.2", food, saturation)
	}
	if got := p.Inventory[player.HotbarStart]; got.ItemID != "minecraft:glass_bottle" || got.Count != 1 {
		t.Fatalf("consumed honey bottle remainder = %+v, want one glass bottle", got)
	}
	if p.UsingItemID != "" || p.UsingItemSlot != -1 || !p.UsingItemSince.IsZero() {
		t.Fatalf("completed food retained active state: %q/%d/%v", p.UsingItemID, p.UsingItemSlot, p.UsingItemSince)
	}
}

func TestBedrockCreativeFoodAnimatesWithoutConsumingStack(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{44}, "bedrock-creative-eater", player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:cooked_chicken", Count: 12}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g}
	s.applyBedrockStartUseItem(intent.StartUseItemIntent{PlayerUUID: p.UUID, HotbarSlot: 0})
	if p.UsingItemID != "minecraft:cooked_chicken" {
		t.Fatalf("creative eating animation did not start: %q", p.UsingItemID)
	}
	p.UsingItemSince = time.Now().Add(-2 * time.Second)
	s.tickBedrockItemUse()
	if got := p.Inventory[player.HotbarStart].Count; got != 12 {
		t.Fatalf("creative food count = %d after use, want 12", got)
	}
	if p.UsingItemID != "" {
		t.Fatalf("creative eating animation did not complete: %q", p.UsingItemID)
	}
}
