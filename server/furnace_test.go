package server

import (
	"testing"

	"GoCraft/core/game"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/core/spatial"
	coreworld "GoCraft/core/world"
	"GoCraft/java/handler"
	"GoCraft/java/session"
)

func newFurnaceTestServer(t *testing.T, blockID string) (*Server, spatial.BlockPos) {
	t.Helper()
	w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	pos := spatial.BlockPos{X: 2, Y: 64, Z: 3}
	w.SetBlock(int(pos.X), int(pos.Y), int(pos.Z), coreworld.Block{
		Namespace: "minecraft", Name: blockID[len("minecraft:"):], Properties: map[string]string{"facing": "north", "lit": "false"},
	})
	s := &Server{
		game: game.New(), world: w, sessions: session.NewManager(),
		furnaces: make(map[furnaceKey]*furnaceState),
	}
	t.Cleanup(func() { _ = w.Close() })
	return s, pos
}

func TestBedrockFurnacesAreIsolatedByDimension(t *testing.T) {
	s, pos := newFurnaceTestServer(t, "minecraft:furnace")
	nether := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
	t.Cleanup(func() { _ = nether.Close() })
	s.netherWorld = nether
	nether.SetBlock(int(pos.X), int(pos.Y), int(pos.Z), bedrockBlock("furnace", map[string]string{"facing": "north", "lit": "false"}))

	overworldState := s.furnaceStateForDimension(dimensionOverworld, pos)
	netherState := s.furnaceStateForDimension(dimensionNether, pos)
	if overworldState == netherState {
		t.Fatal("furnaces at equal coordinates shared state across dimensions")
	}

	p := player.New([16]byte{62}, "nether-smelter", player.ClientEditionBedrock)
	p.Dimension = dimensionNether
	s.openBedrockFurnace(p, pos, "minecraft:furnace")
	p.ContainerSlots[0] = player.ItemStack{ItemID: "minecraft:ancient_debris", Count: 1}
	persistFurnaceSlots(s.worldForPlayer(p), pos, p.OpenContainerKind, p.ContainerSlots)
	if got := nether.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z)); len(got) != 1 || got[0].ItemID != "minecraft:ancient_debris" {
		t.Fatalf("Nether furnace contents = %+v", got)
	}
	if got := s.world.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z)); len(got) != 0 {
		t.Fatalf("Overworld furnace was modified by Nether furnace: %+v", got)
	}
}

func TestBedrockFurnaceTransactionsAcceptInputAndFuelButProtectOutput(t *testing.T) {
	s, pos := newFurnaceTestServer(t, "minecraft:furnace")
	p := player.New([16]byte{61}, "smelter", player.ClientEditionBedrock)
	p.OpenContainerID = 1
	p.OpenContainerKind = "minecraft:furnace"
	p.OpenContainerPos = pos
	p.ContainerSlots = make([]player.ItemStack, 3)
	p.Inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:raw_iron", Count: 2}
	p.Inventory[player.HotbarStart+1] = player.ItemStack{ItemID: "minecraft:coal", Count: 1}
	p.Inventory[player.HotbarStart+2] = player.ItemStack{ItemID: "minecraft:dirt", Count: 1}
	if err := s.game.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	apply := func(action intent.InventoryAction) bool {
		done := make(chan intent.InventoryResult, 1)
		s.applyBedrockInventory(intent.InventoryIntent{PlayerUUID: p.UUID, Actions: []intent.InventoryAction{action}, Done: done})
		return (<-done).Accepted
	}
	if !apply(intent.InventoryAction{Kind: intent.InventoryActionMove, Source: player.HotbarStart, Destination: intent.InventoryFurnaceInput, Count: 2}) {
		t.Fatal("raw iron was rejected from furnace input")
	}
	if !apply(intent.InventoryAction{Kind: intent.InventoryActionMove, Source: player.HotbarStart + 1, Destination: intent.InventoryFurnaceFuel, Count: 1}) {
		t.Fatal("coal was rejected from furnace fuel")
	}
	if apply(intent.InventoryAction{Kind: intent.InventoryActionMove, Source: player.HotbarStart + 2, Destination: intent.InventoryFurnaceOutput, Count: 1}) {
		t.Fatal("ordinary inventory item was accepted into protected furnace output")
	}
	if got := p.ContainerSlots; got[0].ItemID != "minecraft:raw_iron" || got[0].Count != 2 || got[1].ItemID != "minecraft:coal" {
		t.Fatalf("furnace slots after transactions = %+v", got)
	}
	items := s.world.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z))
	if len(items) != 2 {
		t.Fatalf("persisted furnace items = %+v, want input and fuel", items)
	}
}

func TestFurnaceConsumesFuelAndCooksJava1214Recipe(t *testing.T) {
	s, pos := newFurnaceTestServer(t, "minecraft:furnace")
	s.world.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), "minecraft:furnace", []coreworld.ContainerItem{
		{Slot: 0, ItemID: "minecraft:raw_iron", Count: 1},
		{Slot: 1, ItemID: "minecraft:coal", Count: 1},
	})
	state := s.furnaceStateFor(pos)
	s.tickFurnaces()
	if state.BurnDuration != handler.FurnaceFuelDuration("minecraft:coal") || state.BurnTime != state.BurnDuration {
		t.Fatalf("coal burn state = remaining %d/total %d", state.BurnTime, state.BurnDuration)
	}
	if state.CookTime != 1 {
		t.Fatalf("cook progress = %d, want 1", state.CookTime)
	}
	items := s.world.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z))
	for _, item := range items {
		if item.Slot == 1 {
			t.Fatalf("coal was not consumed: %+v", item)
		}
	}
	if s.world.GetBlock(int(pos.X), int(pos.Y), int(pos.Z)).Properties["lit"] != "true" {
		t.Fatal("burning furnace did not switch to lit=true")
	}

	state.CookTime = 199
	s.tickFurnaces()
	items = s.world.ContainerItems(int(pos.X), int(pos.Y), int(pos.Z))
	var input, output coreworld.ContainerItem
	for _, item := range items {
		switch item.Slot {
		case 0:
			input = item
		case 2:
			output = item
		}
	}
	if input.Count != 0 || output.ItemID != "minecraft:iron_ingot" || output.Count != 1 {
		t.Fatalf("furnace contents after cooking: input=%+v output=%+v", input, output)
	}
}

func TestBlastFurnaceBurnsFuelAtDoubleRate(t *testing.T) {
	s, pos := newFurnaceTestServer(t, "minecraft:blast_furnace")
	s.world.SetContainerItems(int(pos.X), int(pos.Y), int(pos.Z), "minecraft:blast_furnace", []coreworld.ContainerItem{
		{Slot: 0, ItemID: "minecraft:raw_iron", Count: 1},
		{Slot: 1, ItemID: "minecraft:coal", Count: 1},
	})
	state := s.furnaceStateFor(pos)
	s.tickFurnaces()
	if want := handler.FurnaceFuelDuration("minecraft:coal") / 2; state.BurnDuration != want {
		t.Fatalf("blast-furnace coal duration = %d, want %d", state.BurnDuration, want)
	}
}
