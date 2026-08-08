package bedrock

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"GoCraft/core/intent"
	"GoCraft/core/player"
	"GoCraft/java/handler"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCanonicalPersonalCraftingSlots(t *testing.T) {
	for bedrockSlot := byte(0); bedrockSlot < 4; bedrockSlot++ {
		got, ok := canonicalInventorySlot(protocol.StackRequestSlotInfo{
			Container: protocol.FullContainerName{ContainerID: protocol.ContainerCraftingInput},
			Slot:      bedrockSlot,
		})
		want := int16(1 + bedrockSlot)
		if !ok || got != want {
			t.Fatalf("local crafting slot %d = %d, %v; want %d, true", bedrockSlot, got, ok, want)
		}
	}
	for bedrockSlot := byte(28); bedrockSlot <= 31; bedrockSlot++ {
		got, ok := canonicalInventorySlot(protocol.StackRequestSlotInfo{
			Container: protocol.FullContainerName{ContainerID: protocol.ContainerCraftingInput},
			Slot:      bedrockSlot,
		})
		want := int16(1 + bedrockSlot - 28)
		if !ok || got != want {
			t.Fatalf("crafting slot %d = %d, %v; want %d, true", bedrockSlot, got, ok, want)
		}
	}
	output, ok := canonicalInventorySlot(protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerCreatedOutput},
		Slot:      50,
	})
	if !ok || output != 0 {
		t.Fatalf("created output = %d, %v; want 0, true", output, ok)
	}
	cursor, ok := canonicalInventorySlot(protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerCursor},
	})
	if !ok || cursor != intent.InventoryCursorSlot {
		t.Fatalf("cursor = %d, %v; want %d, true", cursor, ok, intent.InventoryCursorSlot)
	}
}

func TestCanonicalCraftingTableSlots(t *testing.T) {
	p := player.New([16]byte{}, "crafter", player.ClientEditionBedrock)
	p.OpenContainerKind = "minecraft:crafting_table"
	for bedrockSlot := byte(0); bedrockSlot < 9; bedrockSlot++ {
		got, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
			Container: protocol.FullContainerName{ContainerID: protocol.ContainerCraftingInput}, Slot: bedrockSlot,
		})
		want := intent.InventoryCraftingTableStart + int16(bedrockSlot)
		if !ok || got != want {
			t.Fatalf("local crafting-table slot %d = %d, %v; want %d, true", bedrockSlot, got, ok, want)
		}
	}
	for bedrockSlot := byte(32); bedrockSlot <= 40; bedrockSlot++ {
		got, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
			Container: protocol.FullContainerName{ContainerID: protocol.ContainerCraftingInput}, Slot: bedrockSlot,
		})
		want := intent.InventoryCraftingTableStart + int16(bedrockSlot-32)
		if !ok || got != want {
			t.Fatalf("crafting-table slot %d = %d, %v; want %d, true", bedrockSlot, got, ok, want)
		}
	}
	output, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerCreatedOutput}, Slot: 50,
	})
	if !ok || output != intent.InventoryCraftingTableOutput {
		t.Fatalf("crafting-table output = %d, %v; want %d, true", output, ok, intent.InventoryCraftingTableOutput)
	}
	localOutput, ok := canonicalInventorySlotFor(p, protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerCreatedOutput}, Slot: 0,
	})
	if !ok || localOutput != intent.InventoryCraftingTableOutput {
		t.Fatalf("local crafting-table output = %d, %v; want %d, true", localOutput, ok, intent.InventoryCraftingTableOutput)
	}
}

func TestCraftPredictionActionsAreSuccessfulNoOps(t *testing.T) {
	actions, valid := (&Listener{}).canonicalInventoryActions(nil, []protocol.StackRequestAction{
		&protocol.CraftRecipeStackRequestAction{RecipeNetworkID: 1, NumberOfCrafts: 1},
		&protocol.CreateStackRequestAction{},
		&protocol.CraftResultsDeprecatedStackRequestAction{},
	})
	if !valid || len(actions) != 0 {
		t.Fatalf("craft prediction translated to %d actions, valid=%v; want successful no-op", len(actions), valid)
	}
}

func TestBedrockCraftingDataContainsVanillaRecipes(t *testing.T) {
	if bedrockRecipeCompatibilityVersion != "1.21.4" {
		t.Fatalf("Bedrock recipes target %s, want Java 1.21.4", bedrockRecipeCompatibilityVersion)
	}
	data := bedrockCraftingData()
	javaRecipes := handler.CraftingRecipeCatalog()
	javaRecipeIDs := make(map[string]struct{}, len(javaRecipes))
	javaCraftingRecipes := 0
	craftingTableVariants := 1
	for _, recipe := range javaRecipes {
		javaRecipeIDs[recipe.Name] = struct{}{}
		if recipe.Kind == "shaped" || recipe.Kind == "shapeless" {
			javaCraftingRecipes++
		}
		if recipe.Name == "minecraft:crafting_table" && len(recipe.Ingredients) != 0 {
			craftingTableVariants = len(recipe.Ingredients[0].Alternatives)
			for _, plank := range recipe.Ingredients[0].Alternatives[1:] {
				javaRecipeIDs["gocraft:java_1_21_4/crafting_table/"+strings.TrimPrefix(plank, "minecraft:")] = struct{}{}
			}
		}
	}
	total := len(data.ShapedRecipes) + len(data.ShapelessRecipes) +
		len(data.SmithingTransformRecipes) + len(data.SmithingTrimRecipes)
	expectedTotal := javaCraftingRecipes + craftingTableVariants - 1
	if total != expectedTotal {
		advertised := make(map[string]struct{}, total)
		for _, recipe := range data.ShapedRecipes {
			advertised[recipe.RecipeID] = struct{}{}
		}
		for _, recipe := range data.ShapelessRecipes {
			advertised[recipe.RecipeID] = struct{}{}
		}
		missing := make([]string, 0, javaCraftingRecipes-total)
		for _, recipe := range javaRecipes {
			if recipe.Kind != "shaped" && recipe.Kind != "shapeless" {
				continue
			}
			if _, ok := advertised[recipe.Name]; !ok {
				missing = append(missing, recipe.Name)
			}
		}
		t.Fatalf("published %d protocol variants, want %d Java 1.21.4 recipes plus %d crafting-table plank encodings; missing %s", total, javaCraftingRecipes, craftingTableVariants-1, strings.Join(missing, ", "))
	}
	foundPlanks := false
	seenRecipeIDs := make(map[string]struct{}, total)
	seenNetworkIDs := make(map[uint32]struct{}, total)
	checkIdentity := func(recipeID string, networkID uint32) {
		t.Helper()
		if _, exists := javaRecipeIDs[recipeID]; !exists {
			t.Fatalf("Bedrock advertised non-Java-1.21.4 recipe %s", recipeID)
		}
		if _, exists := seenRecipeIDs[recipeID]; exists {
			t.Fatalf("Bedrock advertised duplicate recipe ID %s", recipeID)
		}
		seenRecipeIDs[recipeID] = struct{}{}
		if networkID == 0 {
			t.Fatalf("recipe %s has network ID zero", recipeID)
		}
		if _, exists := seenNetworkIDs[networkID]; exists {
			t.Fatalf("recipe %s has duplicate network ID %d", recipeID, networkID)
		}
		seenNetworkIDs[networkID] = struct{}{}
	}
	checkUnlock := func(recipeID string, unlock protocol.Optional[protocol.RecipeUnlockRequirement]) {
		t.Helper()
		requirement, present := unlock.Value()
		if !present || requirement.Context != protocol.RecipeUnlockContextAlwaysUnlocked {
			t.Fatalf("recipe %s is missing Pumpkin-compatible AlwaysUnlocked requirement", recipeID)
		}
	}
	for _, recipe := range data.ShapedRecipes {
		checkIdentity(recipe.RecipeID, recipe.RecipeNetworkID)
		checkUnlock(recipe.RecipeID, recipe.UnlockRequirement)
	}
	for _, recipe := range data.ShapelessRecipes {
		checkIdentity(recipe.RecipeID, recipe.RecipeNetworkID)
		checkUnlock(recipe.RecipeID, recipe.UnlockRequirement)
		if recipe.RecipeID == "minecraft:oak_planks" {
			foundPlanks = true
		}
	}
	if !foundPlanks {
		t.Fatal("Bedrock catalogue does not contain minecraft:oak_planks")
	}
	var encoded bytes.Buffer
	data.Marshal(protocol.NewWriter(&encoded, 0))
	if encoded.Len() == 0 {
		t.Fatal("Bedrock crafting data encoded to an empty packet")
	}
	decoded := &packet.CraftingData{}
	decoded.Marshal(protocol.NewReader(bytes.NewBuffer(encoded.Bytes()), 0, true))
	decodedTotal := len(decoded.ShapedRecipes) + len(decoded.ShapelessRecipes) +
		len(decoded.SmithingTransformRecipes) + len(decoded.SmithingTrimRecipes)
	if decodedTotal != total {
		t.Fatalf("crafting-data round trip decoded %d recipes, encoded %d", decodedTotal, total)
	}
}

func TestCraftingTableRecipeAdvertisesEveryJava1214Plank(t *testing.T) {
	var alternatives []string
	for _, recipe := range handler.CraftingRecipeCatalog() {
		if recipe.Name == "minecraft:crafting_table" && len(recipe.Ingredients) != 0 {
			alternatives = recipe.Ingredients[0].Alternatives
			break
		}
	}
	if len(alternatives) == 0 {
		t.Fatal("Java 1.21.4 crafting-table plank alternatives are missing")
	}
	expected := make(map[string]struct{}, len(alternatives))
	for _, plank := range alternatives {
		name, metadata, ok := bedrockItemIdentity(plank)
		if !ok {
			t.Fatalf("plank %s has no Bedrock identity", plank)
		}
		expected[fmt.Sprintf("%s:%d", name, metadata)] = struct{}{}
	}

	data := bedrockCraftingData()
	seen := make(map[string]struct{}, len(expected))
	for _, recipe := range data.ShapedRecipes {
		if recipe.RecipeID != "minecraft:crafting_table" &&
			!strings.HasPrefix(recipe.RecipeID, "gocraft:java_1_21_4/crafting_table/") {
			continue
		}
		if recipe.Width != 2 || recipe.Height != 2 || len(recipe.Input) != 4 {
			t.Fatalf("crafting table shape = %dx%d with %d inputs", recipe.Width, recipe.Height, len(recipe.Input))
		}
		var variant string
		for index, input := range recipe.Input {
			descriptor, ok := input.Descriptor.(*protocol.DefaultItemDescriptor)
			if !ok || descriptor.Name == "" {
				t.Fatalf("input %d descriptor = %#v, want Pumpkin-compatible concrete name", index, input.Descriptor)
			}
			key := fmt.Sprintf("%s:%d", descriptor.Name, descriptor.MetadataValue)
			if index == 0 {
				variant = key
			} else if key != variant {
				t.Fatalf("recipe %s mixes concrete descriptor variants %s and %s", recipe.RecipeID, variant, key)
			}
		}
		if _, ok := expected[variant]; !ok {
			t.Fatalf("recipe %s advertises unexpected plank %s", recipe.RecipeID, variant)
		}
		seen[variant] = struct{}{}
	}
	if len(seen) != len(expected) {
		t.Fatalf("advertised %d crafting-table plank variants, want all %d", len(seen), len(expected))
	}
}

func TestCraftingCatalogueContainsOnlyPumpkinConcreteDescriptors(t *testing.T) {
	data := bedrockCraftingData()
	check := func(recipeID string, inputs []protocol.ItemDescriptorCount) {
		t.Helper()
		for index, input := range inputs {
			switch input.Descriptor.(type) {
			case *protocol.InvalidItemDescriptor, *protocol.DefaultItemDescriptor:
			default:
				t.Fatalf("recipe %s input %d uses non-Pumpkin descriptor %T", recipeID, index, input.Descriptor)
			}
		}
	}
	for _, recipe := range data.ShapedRecipes {
		check(recipe.RecipeID, recipe.Input)
	}
	for _, recipe := range data.ShapelessRecipes {
		check(recipe.RecipeID, recipe.Input)
	}
}

func TestCraftingInputResponseUsesPumpkinChangedSlotsOnly(t *testing.T) {
	p := player.New([16]byte{31}, "crafter", player.ClientEditionBedrock)
	p.Inventory[3] = player.ItemStack{ItemID: "minecraft:birch_log", Count: 1}
	p.Inventory[0] = player.ItemStack{ItemID: "minecraft:birch_planks", Count: 4}
	session := &bedrockSession{nextStackNetworkID: 1}
	action := &protocol.PlaceStackRequestAction{}
	action.Count = 1
	action.Source = protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerCursor},
		Slot:      0,
	}
	action.Destination = protocol.StackRequestSlotInfo{
		Container: protocol.FullContainerName{ContainerID: protocol.ContainerCraftingInput},
		Slot:      30,
	}

	groups := (&Listener{}).stackResponseContainerInfo(session, p, []protocol.StackRequestAction{action})
	for _, group := range groups {
		if group.Container.ContainerID == protocol.ContainerCraftingOutputPreview ||
			group.Container.ContainerID == protocol.ContainerCreatedOutput {
			t.Fatalf("ordinary input move included a virtual output response: %+v", groups)
		}
	}
}

func TestPersonalCraftingSlotPacketsDoNotOverlapHotbar(t *testing.T) {
	p := player.New([16]byte{32}, "crafter", player.ClientEditionBedrock)
	session := &bedrockSession{nextStackNetworkID: 1}
	packets := (&Listener{}).personalCraftingSlotPackets(session, p)
	if len(packets) != 5 {
		t.Fatalf("player crafting packet count = %d, want 5", len(packets))
	}
	for index, pk := range packets {
		container, present := pk.Container.Value()
		if !present || pk.WindowID != protocol.WindowIDUI {
			t.Fatalf("player packet %d = window %d container %+v", index, pk.WindowID, pk.Container)
		}
		wantSlot := uint32(28 + index)
		wantContainer := byte(protocol.ContainerCraftingInput)
		if index == 4 {
			wantSlot = 50
			wantContainer = protocol.ContainerCreatedOutput
		}
		if pk.Slot != wantSlot || container.ContainerID != wantContainer {
			t.Fatalf("player packet %d = slot %d container %d, want slot %d container %d", index, pk.Slot, container.ContainerID, wantSlot, wantContainer)
		}
		if pk.Slot < 9 && container.ContainerID == protocol.ContainerInventory {
			t.Fatalf("player crafting packet %d overlaps Bedrock hotbar slot %d", index, pk.Slot)
		}
	}

	p.OpenContainerKind = "minecraft:crafting_table"
	p.OpenContainerID = 1
	packets = (&Listener{}).personalCraftingSlotPackets(session, p)
	if len(packets) != 10 {
		t.Fatalf("workbench crafting packet count = %d, want 10", len(packets))
	}
	for index, pk := range packets {
		container, present := pk.Container.Value()
		if !present || pk.WindowID != protocol.WindowIDUI {
			t.Fatalf("workbench packet %d = window %d container %+v", index, pk.WindowID, pk.Container)
		}
		if index < 9 {
			if container.ContainerID != protocol.ContainerCraftingInput || pk.Slot != uint32(32+index) {
				t.Fatalf("workbench input packet %d = slot %d container %d", index, pk.Slot, container.ContainerID)
			}
		} else if container.ContainerID != protocol.ContainerCreatedOutput || pk.Slot != 50 {
			t.Fatalf("workbench output packet = slot %d container %d", pk.Slot, container.ContainerID)
		}
	}
}

func TestPlayerScreenKeepsBedrockInventoryOrdering(t *testing.T) {
	want := map[int]int{
		0:  player.HotbarStart,
		8:  player.HotbarStart + 8,
		9:  9,
		35: 35,
	}
	for bedrockSlot, canonical := range want {
		if got := bedrockInventoryCanonicalSlot(bedrockSlot); got != canonical {
			t.Fatalf("Bedrock inventory slot %d = canonical %d, want %d", bedrockSlot, got, canonical)
		}
	}
	for _, invalid := range []int{-1, 36} {
		if got := bedrockInventoryCanonicalSlot(invalid); got != -1 {
			t.Fatalf("invalid Bedrock inventory slot %d = canonical %d, want -1", invalid, got)
		}
	}
}
