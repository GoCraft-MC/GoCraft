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

func TestBedrockInventoryCanEquipArmorAuthoritatively(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{9}, "bedrock", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:iron_helmet", Count: 1}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{
		PlayerUUID: p.UUID,
		Actions: []intent.InventoryAction{{
			Kind: intent.InventoryActionMove, Source: player.HotbarStart, Destination: 5, Count: 1,
		}},
		Done: done,
	})
	if result := <-done; !result.Accepted {
		t.Fatal("valid armour move was rejected")
	}
	if p.Inventory[5].ItemID != "minecraft:iron_helmet" || !p.Inventory[player.HotbarStart].IsEmpty() {
		t.Fatalf("unexpected inventory after equip: helmet=%+v source=%+v", p.Inventory[5], p.Inventory[player.HotbarStart])
	}
}

func TestBedrockChestAcceptsAndPersistsItems(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	pos := spatial.BlockPos{X: 2, Y: 64, Z: 3}
	w.SetBlock(2, 64, 3, coreworld.Block{Namespace: "minecraft", Name: "chest", Properties: map[string]string{
		"facing": "north", "type": "single", "waterlogged": "false",
	}})
	w.SetContainerItems(2, 64, 3, "minecraft:chest", []coreworld.ContainerItem{{
		Slot: 4, ItemID: "minecraft:apple", Count: 3,
	}})

	g := game.New()
	p := player.New([16]byte{41}, "bedrock-chest", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 12}
	_ = g.AddPlayer(p)
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	s.openBedrockGenericContainer(p, pos, "minecraft:chest")
	if len(p.ContainerSlots) != 27 || p.ContainerSlots[4].ItemID != "minecraft:apple" {
		t.Fatalf("loaded chest = %#v", p.ContainerSlots)
	}

	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{PlayerUUID: p.UUID, Actions: []intent.InventoryAction{{
		Kind: intent.InventoryActionMove, Source: player.HotbarStart,
		Destination: intent.InventoryContainerStart + 7, Count: 12,
	}}, Done: done})
	if result := <-done; !result.Accepted {
		t.Fatal("placing an item in the open chest was rejected")
	}
	if !p.Inventory[player.HotbarStart].IsEmpty() || p.ContainerSlots[7].Count != 12 {
		t.Fatalf("chest move did not apply: source=%+v destination=%+v", p.Inventory[player.HotbarStart], p.ContainerSlots[7])
	}

	items := w.ContainerItems(2, 64, 3)
	foundLog := false
	for _, item := range items {
		if item.Slot == 7 && item.ItemID == "minecraft:oak_log" && item.Count == 12 {
			foundLog = true
		}
	}
	if !foundLog {
		t.Fatalf("chest contents were not persisted: %+v", items)
	}
}

func TestBedrockCartographyTableAcceptsInputsAndReturnsThemOnClose(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{44}, "bedrock-cartographer", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:filled_map", Count: 1}
	p.Inventory[player.HotbarStart+1] = player.ItemStack{ItemID: "minecraft:paper", Count: 1}
	_ = g.AddPlayer(p)
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	s.openBedrockWorkstation(p, spatial.BlockPos{X: 1, Y: 64, Z: 1}, "minecraft:cartography_table")

	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{PlayerUUID: p.UUID, Actions: []intent.InventoryAction{
		{Kind: intent.InventoryActionMove, Source: player.HotbarStart, Destination: intent.InventoryContainerStart, Count: 1},
		{Kind: intent.InventoryActionMove, Source: player.HotbarStart + 1, Destination: intent.InventoryContainerStart + 1, Count: 1},
	}, Done: done})
	if result := <-done; !result.Accepted {
		t.Fatal("cartography inputs were rejected")
	}
	if p.ContainerSlots[0].ItemID != "minecraft:filled_map" || p.ContainerSlots[1].ItemID != "minecraft:paper" {
		t.Fatalf("cartography slots = %+v", p.ContainerSlots)
	}

	s.applyBedrockContainerClose(intent.ContainerCloseIntent{PlayerUUID: p.UUID, WindowID: 1})
	if p.OpenContainerKind != "" || len(p.ContainerSlots) != 0 {
		t.Fatalf("cartography screen remained open: %q/%+v", p.OpenContainerKind, p.ContainerSlots)
	}
	foundMap, foundPaper := false, false
	for _, stack := range p.Inventory {
		foundMap = foundMap || stack.ItemID == "minecraft:filled_map" && stack.Count == 1
		foundPaper = foundPaper || stack.ItemID == "minecraft:paper" && stack.Count == 1
	}
	if !foundMap || !foundPaper {
		t.Fatalf("cartography inputs were not returned: map=%v paper=%v inventory=%+v", foundMap, foundPaper, p.Inventory)
	}
}

func TestBedrockInventoryRejectsNonArmorInArmorSlot(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{10}, "bedrock", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:stone", Count: 1}
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{
		PlayerUUID: p.UUID,
		Actions: []intent.InventoryAction{{
			Kind: intent.InventoryActionMove, Source: player.HotbarStart, Destination: 5, Count: 1,
		}},
		Done: done,
	})
	if result := <-done; result.Accepted {
		t.Fatal("stone was accepted as a helmet")
	}
	if p.Inventory[player.HotbarStart].Count != 1 || !p.Inventory[5].IsEmpty() {
		t.Fatal("rejected transaction mutated inventory")
	}
}

func TestBedrockInventoryCursorRoundTrip(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{11}, "bedrock", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[9] = player.ItemStack{ItemID: "minecraft:dirt", Count: 8}
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}

	apply := func(action intent.InventoryAction) {
		t.Helper()
		done := make(chan intent.InventoryResult, 1)
		s.applyBedrockInventory(intent.InventoryIntent{
			PlayerUUID: p.UUID,
			Actions:    []intent.InventoryAction{action},
			Done:       done,
		})
		if result := <-done; !result.Accepted {
			t.Fatalf("inventory action was rejected: %+v", action)
		}
	}

	apply(intent.InventoryAction{
		Kind: intent.InventoryActionMove, Source: 9, Destination: intent.InventoryCursorSlot, Count: 8,
	})
	if !p.Inventory[9].IsEmpty() || p.CarriedItem.ItemID != "minecraft:dirt" || p.CarriedItem.Count != 8 {
		t.Fatalf("dirt was not moved to cursor: source=%+v cursor=%+v", p.Inventory[9], p.CarriedItem)
	}

	apply(intent.InventoryAction{
		Kind: intent.InventoryActionMove, Source: intent.InventoryCursorSlot, Destination: 10, Count: 8,
	})
	if !p.CarriedItem.IsEmpty() || p.Inventory[10].ItemID != "minecraft:dirt" || p.Inventory[10].Count != 8 {
		t.Fatalf("dirt was not moved out of cursor: destination=%+v cursor=%+v", p.Inventory[10], p.CarriedItem)
	}
}

func TestBedrockInventoryCanRecoverInvalidItemFromOffhand(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{21}, "bedrock-offhand-recovery", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.OffhandSlot] = player.ItemStack{ItemID: "minecraft:coal", Count: 7}
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)

	s.applyBedrockInventory(intent.InventoryIntent{
		PlayerUUID: p.UUID,
		Actions: []intent.InventoryAction{{
			Kind:        intent.InventoryActionMove,
			Source:      player.OffhandSlot,
			Destination: 9,
			Count:       7,
		}},
		Done: done,
	})
	if result := <-done; !result.Accepted {
		t.Fatal("moving a stranded stack out of offhand was rejected")
	}
	if !p.Inventory[player.OffhandSlot].IsEmpty() {
		t.Fatalf("offhand remained %+v", p.Inventory[player.OffhandSlot])
	}
	if got := p.Inventory[9]; got.ItemID != "minecraft:coal" || got.Count != 7 {
		t.Fatalf("recovered stack = %+v, want seven coal", got)
	}
}

func TestBedrockCreativeGiveCanBePlacedFromCursor(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{12}, "builder", player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)

	s.applyBedrockInventory(intent.InventoryIntent{
		PlayerUUID: p.UUID,
		Actions: []intent.InventoryAction{
			{
				Kind: intent.InventoryActionCreativeGive, Destination: intent.InventoryCursorSlot,
				Count: 64, Item: player.ItemStack{ItemID: "minecraft:oak_log", Count: 64},
			},
			{
				Kind: intent.InventoryActionMove, Source: intent.InventoryCursorSlot,
				Destination: player.HotbarStart, Count: 64,
			},
		},
		Done: done,
	})
	if result := <-done; !result.Accepted {
		t.Fatal("creative give and placement was rejected")
	}
	if !p.CarriedItem.IsEmpty() || p.Inventory[player.HotbarStart].ItemID != "minecraft:oak_log" || p.Inventory[player.HotbarStart].Count != 64 {
		t.Fatalf("creative log did not reach hotbar: hotbar=%+v cursor=%+v", p.Inventory[player.HotbarStart], p.CarriedItem)
	}
}

func TestBedrockPersonalCraftingProducesAndConsumesRecipe(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{13}, "crafter", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 2}
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}

	apply := func(action intent.InventoryAction) {
		t.Helper()
		done := make(chan intent.InventoryResult, 1)
		s.applyBedrockInventory(intent.InventoryIntent{
			PlayerUUID: p.UUID,
			Actions:    []intent.InventoryAction{action},
			Done:       done,
		})
		if result := <-done; !result.Accepted {
			t.Fatalf("inventory action was rejected: %+v", action)
		}
	}

	apply(intent.InventoryAction{
		Kind: intent.InventoryActionMove, Source: player.HotbarStart, Destination: 1, Count: 2,
	})
	if p.Inventory[0].ItemID != "minecraft:oak_planks" || p.Inventory[0].Count != 4 {
		t.Fatalf("crafting output = %+v, want four oak planks", p.Inventory[0])
	}

	apply(intent.InventoryAction{
		Kind: intent.InventoryActionMove, Source: 0, Destination: intent.InventoryCursorSlot, Count: 4,
	})
	if p.CarriedItem.ItemID != "minecraft:oak_planks" || p.CarriedItem.Count != 4 {
		t.Fatalf("cursor = %+v, want four oak planks", p.CarriedItem)
	}
	if p.Inventory[1].Count != 1 {
		t.Fatalf("remaining log count = %d, want 1", p.Inventory[1].Count)
	}
	if p.Inventory[0].ItemID != "minecraft:oak_planks" || p.Inventory[0].Count != 4 {
		t.Fatalf("next crafting output = %+v, want four oak planks", p.Inventory[0])
	}
}

func TestBedrockAutoCraftConsumesAllRequestedIngredients(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{43}, "auto-crafter", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Inventory[1] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 8}
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{PlayerUUID: p.UUID, Actions: []intent.InventoryAction{{
		Kind: intent.InventoryActionMove, Source: 0, Destination: intent.InventoryCursorSlot,
		Count: 32, CraftCount: 8,
	}}, Done: done})
	if result := <-done; !result.Accepted {
		t.Fatal("valid eight-batch Bedrock craft was rejected")
	}
	if !p.Inventory[1].IsEmpty() || !p.Inventory[0].IsEmpty() {
		t.Fatalf("crafting inputs were not exhausted: input=%+v output=%+v", p.Inventory[1], p.Inventory[0])
	}
	if p.CarriedItem.ItemID != "minecraft:oak_planks" || p.CarriedItem.Count != 32 {
		t.Fatalf("crafted cursor stack = %+v, want 32 oak planks", p.CarriedItem)
	}
}

func TestBedrockPersonalCraftingAcceptsEveryJava1214Plank(t *testing.T) {
	planks := []string{
		"minecraft:oak_planks", "minecraft:spruce_planks", "minecraft:birch_planks",
		"minecraft:jungle_planks", "minecraft:acacia_planks", "minecraft:cherry_planks",
		"minecraft:dark_oak_planks", "minecraft:pale_oak_planks", "minecraft:mangrove_planks",
		"minecraft:bamboo_planks", "minecraft:crimson_planks", "minecraft:warped_planks",
	}
	for _, plank := range planks {
		t.Run(plank, func(t *testing.T) {
			g := game.New()
			p := player.New([16]byte{16}, "crafter", player.ClientEditionBedrock)
			p.GameMode = player.GameModeSurvival
			for slot := 1; slot <= 3; slot++ {
				p.Inventory[slot] = player.ItemStack{ItemID: plank, Count: 1}
			}
			p.CarriedItem = player.ItemStack{ItemID: plank, Count: 1}
			_ = g.AddPlayer(p)
			s := &Server{game: g, sessions: session.NewManager()}
			done := make(chan intent.InventoryResult, 1)
			s.applyBedrockInventory(intent.InventoryIntent{
				PlayerUUID: p.UUID,
				Actions: []intent.InventoryAction{{
					Kind: intent.InventoryActionMove, Source: intent.InventoryCursorSlot, Destination: 4, Count: 1,
				}},
				Done: done,
			})
			if result := <-done; !result.Accepted {
				t.Fatal("placing the fourth plank was rejected")
			}
			if result := p.Inventory[0]; result.ItemID != "minecraft:crafting_table" || result.Count != 1 {
				t.Fatalf("result = %+v, want one crafting table", result)
			}
		})
	}
}

func TestBedrockCraftingTableProducesAndConsumesRecipe(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{14}, "table-crafter", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.OpenContainerKind = "minecraft:crafting_table"
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 1}
	_ = g.AddPlayer(p)
	s := &Server{game: g, sessions: session.NewManager()}

	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{PlayerUUID: p.UUID, Actions: []intent.InventoryAction{{
		Kind: intent.InventoryActionMove, Source: player.HotbarStart,
		Destination: intent.InventoryCraftingTableStart, Count: 1,
	}}, Done: done})
	if result := <-done; !result.Accepted {
		t.Fatal("placing a log in the crafting table was rejected")
	}
	if p.CraftingResult.ItemID != "minecraft:oak_planks" || p.CraftingResult.Count != 4 {
		t.Fatalf("crafting result = %+v, want four oak planks", p.CraftingResult)
	}

	done = make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{PlayerUUID: p.UUID, Actions: []intent.InventoryAction{{
		Kind: intent.InventoryActionMove, Source: intent.InventoryCraftingTableOutput,
		Destination: intent.InventoryCursorSlot, Count: 4,
	}}, Done: done})
	if result := <-done; !result.Accepted {
		t.Fatal("taking the crafting-table result was rejected")
	}
	if p.CarriedItem.ItemID != "minecraft:oak_planks" || p.CarriedItem.Count != 4 || !p.CraftingGrid[0].IsEmpty() {
		t.Fatalf("cursor=%+v input=%+v", p.CarriedItem, p.CraftingGrid[0])
	}
}

func TestBedrockCraftingTableCloseClearsStateAndReturnsIngredients(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{15}, "closing-crafter", player.ClientEditionBedrock)
	p.OpenContainerID = 1
	p.OpenContainerKind = "minecraft:crafting_table"
	p.CraftingGrid[4] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 1}
	_ = g.AddPlayer(p)
	s := &Server{game: g}

	// Bedrock may report a protocol-specific window ID while closing. Closing
	// the active server-side container must not leave the player stuck in the
	// 3x3 crafting context.
	s.applyBedrockContainerClose(intent.ContainerCloseIntent{PlayerUUID: p.UUID, WindowID: 0xff})
	if p.OpenContainerID != 0 || p.OpenContainerKind != "" {
		t.Fatalf("container state remained open: id=%d kind=%q", p.OpenContainerID, p.OpenContainerKind)
	}
	if !p.CraftingGrid[4].IsEmpty() {
		t.Fatalf("crafting input was not returned: %+v", p.CraftingGrid[4])
	}
	if got := p.Inventory[player.HotbarStart]; got.ItemID != "minecraft:oak_log" || got.Count != 1 {
		t.Fatalf("returned inventory item = %+v, want oak log", got)
	}
}

func TestBedrockInventoryDropRemovesItemsAndSpawnsEntity(t *testing.T) {
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	defer w.Close()
	g := game.New()
	p := player.New([16]byte{75}, "bedrock-dropper", player.ClientEditionBedrock)
	p.GameMode = player.GameModeSurvival
	p.Position = spatial.Vec3{X: 3, Y: 65, Z: 4}
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:diamond", Count: 5}
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	s := &Server{game: g, world: w, sessions: session.NewManager()}
	done := make(chan intent.InventoryResult, 1)
	s.applyBedrockInventory(intent.InventoryIntent{
		PlayerUUID: p.UUID,
		Actions: []intent.InventoryAction{{
			Kind: intent.InventoryActionDrop, Source: player.HotbarStart, Count: 2,
		}},
		Done: done,
	})
	if result := <-done; !result.Accepted {
		t.Fatal("valid Bedrock drop was rejected")
	}
	if got := p.Inventory[player.HotbarStart]; got.ItemID != "minecraft:diamond" || got.Count != 3 {
		t.Fatalf("source after drop = %+v, want three diamonds", got)
	}
	entities := w.Entities.Snapshot()
	if len(entities) != 1 || entities[0].ItemID != "minecraft:diamond" || entities[0].ItemCount != 2 {
		t.Fatalf("dropped entities = %+v", entities)
	}
}
