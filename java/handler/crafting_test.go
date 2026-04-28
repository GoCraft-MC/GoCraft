package handler

import (
	"testing"

	"GoCraft/core/player"
)

func TestRecipePlacementUsesInventoryAndProducesCraftingResult(t *testing.T) {
	var stick recipeDisplay
	found := false
	for _, recipe := range javaRecipeDisplays {
		if recipe.name == "minecraft:stick" {
			stick, found = recipe, true
			break
		}
	}
	if !found {
		t.Fatal("complete catalog does not contain minecraft:stick")
	}
	template, err := craftingTemplate(stick)
	if err != nil {
		t.Fatal(err)
	}
	var inventory [player.InventorySize]player.ItemStack
	inventory[player.HotbarStart] = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 2}
	var grid [9]player.ItemStack
	if !placeRecipeOnce(&inventory, &grid, template) {
		t.Fatal("automatic placement rejected two oak planks for the stick recipe")
	}
	if inventory[player.HotbarStart].Count != 0 || grid[0].ItemID != "minecraft:oak_planks" || grid[3].ItemID != "minecraft:oak_planks" {
		t.Fatalf("placement inventory=%+v grid[0]=%+v grid[3]=%+v", inventory[player.HotbarStart], grid[0], grid[3])
	}
	if result := findCraftingResult(grid); result.ItemID != "minecraft:stick" || result.Count != 4 {
		t.Fatalf("crafting result = %+v, want four sticks", result)
	}
}

func TestTakingCraftingResultConsumesOneIngredientPerOccupiedSlot(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionJava)
	p.CraftingGrid[0] = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 2}
	p.CraftingGrid[3] = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 2}
	p.CraftingResult = player.ItemStack{ItemID: "minecraft:stick", Count: 4}
	takeCraftingResult(p)
	if p.CarriedItem.ItemID != "minecraft:stick" || p.CarriedItem.Count != 4 {
		t.Fatalf("cursor = %+v, want four sticks", p.CarriedItem)
	}
	if p.CraftingGrid[0].Count != 1 || p.CraftingGrid[3].Count != 1 {
		t.Fatalf("remaining grid counts = %d/%d, want 1/1", p.CraftingGrid[0].Count, p.CraftingGrid[3].Count)
	}
	if next := findCraftingResult(p.CraftingGrid); next.ItemID != "minecraft:stick" || next.Count != 4 {
		t.Fatalf("next result = %+v, want another four sticks", next)
	}
}

func TestPersonalCraftingResolvesWoodVariantsToMatchingPlanks(t *testing.T) {
	tests := []struct {
		input string
		want  string
		count int
	}{
		{"minecraft:oak_log", "minecraft:oak_planks", 4},
		{"minecraft:birch_log", "minecraft:birch_planks", 4},
		{"minecraft:stripped_spruce_wood", "minecraft:spruce_planks", 4},
		{"minecraft:mangrove_log", "minecraft:mangrove_planks", 4},
		{"minecraft:crimson_stem", "minecraft:crimson_planks", 4},
		{"minecraft:bamboo_block", "minecraft:bamboo_planks", 2},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			grid := [4]player.ItemStack{{ItemID: test.input, Count: 1}}
			got := FindPersonalCraftingResult(grid)
			if got.ItemID != test.want || got.Count != test.count {
				t.Fatalf("result = %+v, want %d %s", got, test.count, test.want)
			}
		})
	}
}

func TestPersonalCraftingSupportsShapedTwoByTwoRecipes(t *testing.T) {
	grid := [4]player.ItemStack{
		{ItemID: "minecraft:oak_planks", Count: 1},
		{ItemID: "minecraft:oak_planks", Count: 1},
		{ItemID: "minecraft:oak_planks", Count: 1},
		{ItemID: "minecraft:oak_planks", Count: 1},
	}
	got := FindPersonalCraftingResult(grid)
	if got.ItemID != "minecraft:crafting_table" || got.Count != 1 {
		t.Fatalf("result = %+v, want crafting table", got)
	}
}

func TestMaceRecipeWorksInEveryCraftingColumn(t *testing.T) {
	for column := 0; column < 3; column++ {
		var grid [9]player.ItemStack
		grid[column] = player.ItemStack{ItemID: "minecraft:heavy_core", Count: 1}
		grid[3+column] = player.ItemStack{ItemID: "minecraft:breeze_rod", Count: 1}
		got := FindCraftingTableResult(grid)
		if got.ItemID != "minecraft:mace" || got.Count != 1 {
			t.Fatalf("column %d result = %+v, want one mace", column, got)
		}
	}
}

func TestPumpkinBedrockCopperArmorRecipesProducePristineStacks(t *testing.T) {
	tests := []struct {
		item       string
		durability int
		armor      int
		pattern    [3]string
	}{
		{"minecraft:copper_helmet", 121, 2, [3]string{"XXX", "X X"}},
		{"minecraft:copper_chestplate", 176, 4, [3]string{"X X", "XXX", "XXX"}},
		{"minecraft:copper_leggings", 165, 3, [3]string{"XXX", "X X", "X X"}},
		{"minecraft:copper_boots", 143, 1, [3]string{"X X", "X X"}},
	}

	if got := len(PumpkinBedrockCraftingRecipeCatalog()); got != len(tests) {
		t.Fatalf("Pumpkin Bedrock recipe supplement has %d recipes, want %d", got, len(tests))
	}
	for _, test := range tests {
		t.Run(test.item, func(t *testing.T) {
			var grid [9]player.ItemStack
			for y, row := range test.pattern {
				for x, symbol := range row {
					if symbol == 'X' {
						grid[y*3+x] = player.ItemStack{ItemID: "minecraft:copper_ingot", Count: 1}
					}
				}
			}
			if javaResult := FindCraftingTableResult(grid); !javaResult.IsEmpty() {
				t.Fatalf("Java 1.21.4 resolver exposed future recipe result %+v", javaResult)
			}
			result := FindBedrockCraftingTableResult(grid)
			if result.ItemID != test.item || result.Count != 1 || result.Damage != 0 {
				t.Fatalf("Pumpkin recipe result = %+v, want one pristine %s", result, test.item)
			}
			if got := player.MaxDurability(result.ItemID); got != test.durability {
				t.Errorf("crafted durability = %d, want %d", got, test.durability)
			}
			if got := player.ArmorPoints(result.ItemID); got != test.armor {
				t.Errorf("crafted armor = %d, want %d", got, test.armor)
			}
		})
	}
}

func TestPlayerInventoryCraftingClickProducesAndConsumesPlanks(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionJava)
	p.CarriedItem = player.ItemStack{ItemID: "minecraft:oak_log", Count: 2}

	clickPlayerInventorySlot(p, 3, 0)
	updatePersonalCraftingResult(p)
	if got := p.Inventory[0]; got.ItemID != "minecraft:oak_planks" || got.Count != 4 {
		t.Fatalf("crafting output = %+v, want four oak planks", got)
	}

	takePersonalCraftingResult(p)
	updatePersonalCraftingResult(p)
	if p.CarriedItem.ItemID != "minecraft:oak_planks" || p.CarriedItem.Count != 4 {
		t.Fatalf("cursor = %+v, want four oak planks", p.CarriedItem)
	}
	if p.Inventory[3].Count != 1 {
		t.Fatalf("remaining logs = %d, want 1", p.Inventory[3].Count)
	}
	if got := p.Inventory[0]; got.ItemID != "minecraft:oak_planks" || got.Count != 4 {
		t.Fatalf("next crafting output = %+v, want four oak planks", got)
	}
}

func TestShiftClickPersonalCraftingInputReturnsItToInventory(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionJava)
	p.Inventory[2] = player.ItemStack{ItemID: "minecraft:birch_log", Count: 1}
	updatePersonalCraftingResult(p)

	shiftPlayerInventorySlot(p, 2)
	updatePersonalCraftingResult(p)
	if !p.Inventory[2].IsEmpty() || !p.Inventory[0].IsEmpty() {
		t.Fatalf("crafting slots not cleared: input=%+v output=%+v", p.Inventory[2], p.Inventory[0])
	}
	if got := p.Inventory[player.HotbarStart]; got.ItemID != "minecraft:birch_log" || got.Count != 1 {
		t.Fatalf("hotbar = %+v, want birch log", got)
	}
}

func TestShiftClickPersonalCraftingResultCraftsMaximum(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionJava)
	p.Inventory[1] = player.ItemStack{ItemID: "minecraft:oak_log", Count: 8}
	updatePersonalCraftingResult(p)

	shiftPersonalCraftingResult(p)

	if !p.Inventory[1].IsEmpty() || !p.Inventory[0].IsEmpty() {
		t.Fatalf("crafting grid was not exhausted: input=%+v output=%+v", p.Inventory[1], p.Inventory[0])
	}
	total := 0
	for slot := 9; slot < player.InventorySize; slot++ {
		if p.Inventory[slot].ItemID == "minecraft:oak_planks" {
			total += p.Inventory[slot].Count
		}
	}
	if total != 32 {
		t.Fatalf("shift-crafted %d planks, want 32", total)
	}
}

func TestShiftClickCraftingTableResultCraftsMaximum(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionJava)
	p.CraftingGrid[0] = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 7}
	p.CraftingGrid[3] = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 7}
	p.CraftingResult = findCraftingResult(p.CraftingGrid)

	shiftCraftingResult(p)

	if !p.CraftingGrid[0].IsEmpty() || !p.CraftingGrid[3].IsEmpty() || !p.CraftingResult.IsEmpty() {
		t.Fatalf("crafting table was not exhausted: grid=%+v result=%+v", p.CraftingGrid, p.CraftingResult)
	}
	total := 0
	for slot := 9; slot < player.InventorySize; slot++ {
		if p.Inventory[slot].ItemID == "minecraft:stick" {
			total += p.Inventory[slot].Count
		}
	}
	if total != 28 {
		t.Fatalf("shift-crafted %d sticks, want 28", total)
	}
}

func TestQuickCraftDragPlacesItemsInCraftingTable(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionJava)
	p.CarriedItem = player.ItemStack{ItemID: "minecraft:oak_planks", Count: 2}
	target := func(slot int) *player.ItemStack { return craftingContainerSlot(p, slot) }

	handleQuickCraft(p, -999, 0, target)
	handleQuickCraft(p, 1, 1, target)
	handleQuickCraft(p, 4, 1, target)
	handleQuickCraft(p, -999, 2, target)

	if p.CraftingGrid[0].Count != 1 || p.CraftingGrid[3].Count != 1 || !p.CarriedItem.IsEmpty() {
		t.Fatalf("drag result grid=%+v/%+v cursor=%+v", p.CraftingGrid[0], p.CraftingGrid[3], p.CarriedItem)
	}
	if result := findCraftingResult(p.CraftingGrid); result.ItemID != "minecraft:stick" || result.Count != 4 {
		t.Fatalf("crafting result = %+v, want four sticks", result)
	}
}
