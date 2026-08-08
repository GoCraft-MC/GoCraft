package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
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
