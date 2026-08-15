package handler

import (
	"testing"

	"GoCraft/core/player"
	"GoCraft/core/spatial"
)

func TestWorkstationLayoutsMatchVanillaScreens(t *testing.T) {
	tests := map[string]struct {
		slots, output int
	}{
		"minecraft:anvil":             {3, 2},
		"minecraft:enchanting_table":  {2, -1},
		"minecraft:grindstone":        {3, 2},
		"minecraft:loom":              {4, 3},
		"minecraft:smithing_table":    {4, 3},
		"minecraft:stonecutter":       {2, 1},
		"minecraft:brewing_stand":     {5, -1},
		"minecraft:cartography_table": {3, 2},
		"minecraft:beacon":            {1, -1},
	}
	for kind, want := range tests {
		if !IsWorkstation(kind) || WorkstationSlotCount(kind) != want.slots || WorkstationOutputIndex(kind) != want.output {
			t.Errorf("%s layout = workstation %v, slots %d, output %d; want true, %d, %d",
				kind, IsWorkstation(kind), WorkstationSlotCount(kind), WorkstationOutputIndex(kind), want.slots, want.output)
		}
	}
}

func TestSmithingOutputConsumesOnlyMatchingProfessionSlots(t *testing.T) {
	slots := []player.ItemStack{
		{ItemID: "minecraft:netherite_upgrade_smithing_template", Count: 2},
		{ItemID: "minecraft:diamond_pickaxe", Count: 1, Damage: 125},
		{ItemID: "minecraft:netherite_ingot", Count: 3},
		{},
	}
	UpdateWorkstationResult("minecraft:smithing_table", slots, 0)
	if got := slots[3]; got.ItemID != "minecraft:netherite_pickaxe" || got.Count != 1 || got.Damage != 125 {
		t.Fatalf("smithing preview = %+v", got)
	}
	result, ok := TakeWorkstationResult("minecraft:smithing_table", slots, 0)
	if !ok || result.ItemID != "minecraft:netherite_pickaxe" {
		t.Fatalf("smithing take = %+v, %v", result, ok)
	}
	if slots[0].Count != 1 || !slots[1].IsEmpty() || slots[2].Count != 2 || !slots[3].IsEmpty() {
		t.Fatalf("smithing ingredients after take = %+v", slots)
	}
}

func TestJavaWorkstationOutputIsAuthoritativeAndInputsReturn(t *testing.T) {
	p := player.New([16]byte{81}, "smith", player.ClientEditionJava)
	if err := openWorkstation(p, nil, nil, spatial.BlockPos{X: 2, Y: 64, Z: 3}, "minecraft:smithing_table"); err != nil {
		t.Fatal(err)
	}
	p.ContainerSlots[0] = player.ItemStack{ItemID: "minecraft:netherite_upgrade_smithing_template", Count: 1}
	p.ContainerSlots[1] = player.ItemStack{ItemID: "minecraft:diamond_sword", Count: 1, Damage: 10}
	p.ContainerSlots[2] = player.ItemStack{ItemID: "minecraft:netherite_ingot", Count: 1}
	UpdateWorkstationResult(p.OpenContainerKind, p.ContainerSlots, p.WorkstationSelection)
	handleWorkstationClick(p, 3, 0, 0)
	if p.CarriedItem.ItemID != "minecraft:netherite_sword" || p.CarriedItem.Damage != 10 {
		t.Fatalf("cursor after output click = %+v", p.CarriedItem)
	}
	if !p.ContainerSlots[0].IsEmpty() || !p.ContainerSlots[1].IsEmpty() || !p.ContainerSlots[2].IsEmpty() {
		t.Fatalf("smithing inputs were not consumed: %+v", p.ContainerSlots)
	}

	p.CarriedItem = player.ItemStack{}
	p.ContainerSlots[0] = player.ItemStack{ItemID: "minecraft:netherite_upgrade_smithing_template", Count: 1}
	closeWorkstation(p, nil)
	if p.OpenContainerKind != "" || p.ContainerSlots != nil {
		t.Fatalf("workstation remained open: %q/%+v", p.OpenContainerKind, p.ContainerSlots)
	}
	found := false
	for _, stack := range p.Inventory {
		found = found || stack.ItemID == "minecraft:netherite_upgrade_smithing_template"
	}
	if !found {
		t.Fatal("workstation input was not returned on close")
	}
}

func TestStonecutterUsesPublishedVanillaRecipeSelection(t *testing.T) {
	slots := []player.ItemStack{{ItemID: "minecraft:cobblestone", Count: 2}, {}}
	UpdateWorkstationResult("minecraft:stonecutter", slots, 0)
	if slots[1].IsEmpty() || slots[1].ItemID == "minecraft:cobblestone" {
		t.Fatalf("stonecutter preview = %+v", slots[1])
	}
	if _, ok := TakeWorkstationResult("minecraft:stonecutter", slots, 0); !ok || slots[0].Count != 1 {
		t.Fatalf("stonecutter did not consume one input: %+v", slots)
	}
}
