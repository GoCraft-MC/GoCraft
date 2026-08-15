package bedrock

import (
	"bytes"
	"strconv"
	"testing"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestPumpkinItemRegistryCoversCreativeCatalogue(t *testing.T) {
	registry := bedrockItemRegistry()
	if len(registry) != 1976 {
		t.Fatalf("item registry has %d entries, want Pumpkin's complete 1976", len(registry))
	}
	byRuntimeID := make(map[int32]string, len(registry))
	byName := make(map[string]protocol.ItemEntry, len(registry))
	for _, entry := range registry {
		if previous, exists := byRuntimeID[int32(entry.RuntimeID)]; exists {
			t.Fatalf("runtime ID %d is shared by %s and %s", entry.RuntimeID, previous, entry.Name)
		}
		if _, exists := byName[entry.Name]; exists {
			t.Fatalf("duplicate item registry name %s", entry.Name)
		}
		byRuntimeID[int32(entry.RuntimeID)] = entry.Name
		byName[entry.Name] = entry
	}
	for _, creative := range pumpkinCreativeItems {
		if name := byRuntimeID[creative.runtimeID]; name != creative.name {
			t.Errorf("creative %s uses runtime ID %d registered as %q", creative.name, creative.runtimeID, name)
		}
	}
	if apple, ok := byName["minecraft:apple"]; !ok {
		t.Fatal("item registry does not contain minecraft:apple")
	} else if apple.Version == protocol.ItemEntryVersionDataDriven && len(apple.Data) == 0 {
		t.Fatal("data-driven apple has no item components")
	}
}

func TestPumpkinItemRegistryMarshalsAsBedrockPacket(t *testing.T) {
	var payload bytes.Buffer
	pk := &packet.ItemRegistry{Items: bedrockItemRegistry()}
	pk.Marshal(protocol.NewWriter(&payload, 0))
	if payload.Len() < 100_000 {
		t.Fatalf("encoded ItemRegistry is only %d bytes; complete table was not written", payload.Len())
	}
}

func TestCreativeCatalogueIsPopulated(t *testing.T) {
	l := &Listener{}
	l.initCreativeContent()
	if len(l.creativeGroups) == 0 || len(l.creativeItems) < 1000 {
		t.Fatalf("creative catalogue = %d groups/%d items", len(l.creativeGroups), len(l.creativeItems))
	}
	for _, item := range l.creativeNames {
		if item.name == "minecraft:oak_log" {
			return
		}
	}
	t.Fatal("creative catalogue does not contain minecraft:oak_log")
}

func TestCreativeCataloguePreservesCurrentPumpkinVariantData(t *testing.T) {
	l := &Listener{}
	l.initCreativeContent()
	if len(l.creativeGroups) != 123 {
		t.Fatalf("creative group count = %d, want Pumpkin's 123", len(l.creativeGroups))
	}
	var nbtVariants, blockVariants int
	for _, item := range l.creativeItems {
		if item.GroupIndex >= uint32(len(l.creativeGroups)) {
			t.Fatalf("item %d has invalid group %d", item.CreativeItemNetworkID, item.GroupIndex)
		}
		if len(item.Item.NBTData) != 0 {
			nbtVariants++
		}
		if item.Item.BlockRuntimeID != 0 {
			blockVariants++
		}
	}
	if nbtVariants != 175 || blockVariants != 939 {
		t.Fatalf("preserved creative variants = %d NBT/%d block states, want 175/939", nbtVariants, blockVariants)
	}
}

func TestCreativeEnchantedBooksKeepDistinctNBTForSearchIndex(t *testing.T) {
	l := &Listener{}
	l.initCreativeContent()
	encoded := make(map[string]struct{})
	books := 0
	for _, item := range l.creativeItems {
		if item.Item.NetworkID != pumpkinCreativeRuntimeIDsByName["minecraft:enchanted_book"] {
			continue
		}
		books++
		if len(item.Item.NBTData) == 0 {
			t.Fatalf("enchanted book %d has no NBT variant", item.CreativeItemNetworkID)
		}
		var payload bytes.Buffer
		if err := nbt.NewEncoderWithEncoding(&payload, nbt.LittleEndian).Encode(item.Item.NBTData); err != nil {
			t.Fatal(err)
		}
		encoded[payload.String()] = struct{}{}
	}
	if books != 125 || len(encoded) != books {
		t.Fatalf("enchanted book variants = %d entries/%d distinct NBT, want 125/125", books, len(encoded))
	}
}

func TestCanonicalLightItemUsesSelectedBedrockBlockState(t *testing.T) {
	encoder := bedrockworld.NewEncoder()
	l := &Listener{encoder: encoder}
	instance := l.itemInstance(player.ItemStack{ItemID: "minecraft:light", Count: 1, Damage: 11}, 7)
	wantBlock := encoder.BlockNetworkID(coreworld.Block{
		Namespace: "minecraft", Name: "light", Properties: map[string]string{"level": "11"},
	})
	if instance.Stack.NetworkID == 0 || instance.Stack.BlockRuntimeID != int32(wantBlock) {
		t.Fatalf("light item = network %d block %d, want nonzero/%d", instance.Stack.NetworkID, instance.Stack.BlockRuntimeID, wantBlock)
	}
	if instance.Stack.NBTData != nil {
		t.Fatalf("light level was encoded as durability NBT: %v", instance.Stack.NBTData)
	}
}

func TestBedrockLightCreativeItemsUseCanonicalJavaIdentity(t *testing.T) {
	for level := int16(0); level <= 15; level++ {
		name, meta := canonicalCreativeIdentity("minecraft:light_block_"+strconv.Itoa(int(level)), 0)
		if name != "minecraft:light" || meta != level {
			t.Fatalf("level %d canonical identity = %q/%d", level, name, meta)
		}
	}
}

func TestBedrockLegacyCreativeDoorUsesCanonicalOakDoor(t *testing.T) {
	name, meta := canonicalCreativeIdentity("minecraft:wooden_door", 0)
	if name != "minecraft:oak_door" || meta != 0 {
		t.Fatalf("wooden door canonical identity = %q/%d, want minecraft:oak_door/0", name, meta)
	}
}

func TestBedrockCreativeToolBecomesPristineCanonicalStack(t *testing.T) {
	l := &Listener{encoder: bedrockworld.NewEncoder()}
	l.initCreativeContent()
	var creativeID uint32
	var advertised protocol.ItemStack
	for id, item := range l.creativeNames {
		if item.name == "minecraft:iron_pickaxe" {
			creativeID = id
			break
		}
	}
	if creativeID == 0 {
		t.Fatal("creative catalogue does not contain minecraft:iron_pickaxe")
	}
	for _, item := range l.creativeItems {
		if item.CreativeItemNetworkID == creativeID {
			advertised = item.Item
			break
		}
	}
	mapping := javaToBedrockItemMappings["minecraft:iron_pickaxe"]
	if advertised.NetworkID != mapping.runtimeID || advertised.MetadataValue != mapping.metadata {
		t.Fatalf("advertised iron pickaxe identity = %d/%d, canonical inventory identity = %d/%d",
			advertised.NetworkID, advertised.MetadataValue, mapping.runtimeID, mapping.metadata)
	}

	p := player.New([16]byte{31}, "creative-miner", player.ClientEditionBedrock)
	p.GameMode = player.GameModeCreative
	actions, ok := l.canonicalInventoryActions(p, []protocol.StackRequestAction{
		&protocol.CraftCreativeStackRequestAction{
			CreativeItemNetworkID: creativeID,
			NumberOfCrafts:        1,
		},
	})
	if !ok || len(actions) != 1 || actions[0].Kind != intent.InventoryActionCreativeGive {
		t.Fatalf("creative request = accepted %v, actions %+v", ok, actions)
	}
	got := actions[0].Item
	if got.ItemID != "minecraft:iron_pickaxe" || got.Count != 1 || got.Damage != 0 {
		t.Fatalf("creative iron pickaxe = %+v, want pristine canonical tool", got)
	}
	if player.MaxDurability(got.ItemID) != 250 {
		t.Fatalf("creative iron pickaxe durability = %d, want 250", player.MaxDurability(got.ItemID))
	}
	inventoryItem := l.itemInstance(got, 1).Stack
	if inventoryItem.NetworkID != advertised.NetworkID || inventoryItem.MetadataValue != advertised.MetadataValue {
		t.Fatalf("creative iron pickaxe changed identity in inventory: advertised %d/%d, inventory %d/%d",
			advertised.NetworkID, advertised.MetadataValue, inventoryItem.NetworkID, inventoryItem.MetadataValue)
	}
}

func TestOrdinaryCreativeAuxValueIsNotDurability(t *testing.T) {
	name, damage := canonicalCreativeIdentity("minecraft:iron_pickaxe", 87)
	if name != "minecraft:iron_pickaxe" || damage != 0 {
		t.Fatalf("canonical identity = %q/%d, want pristine iron pickaxe", name, damage)
	}
}

func TestAllMappedBedrockCreativeItemsUsePumpkinRuntimePalette(t *testing.T) {
	l := &Listener{encoder: bedrockworld.NewEncoder()}
	l.initCreativeContent()

	advertised := make(map[uint32]protocol.ItemStack, len(l.creativeItems))
	for _, item := range l.creativeItems {
		advertised[item.CreativeItemNetworkID] = item.Item
	}

	checked := 0
	for creativeID, known := range l.creativeNames {
		if known.name == "minecraft:light" {
			// Light is the one creative item whose level is carried by its
			// metadata, so it intentionally has special handling.
			continue
		}
		if _, ok := javaToBedrockItemMappings[known.name]; !ok {
			continue
		}

		creative, ok := advertised[creativeID]
		if !ok {
			t.Fatalf("creative item %d (%s) has no advertised stack", creativeID, known.name)
		}
		inventory := l.itemInstance(player.ItemStack{
			ItemID: known.name,
			Count:  1,
			Damage: int(known.meta),
		}, 1).Stack
		if creative.NetworkID != inventory.NetworkID {
			t.Errorf(
				"creative item %d (%s) advertises runtime ID %d, normal inventory uses %d",
				creativeID,
				known.name,
				creative.NetworkID,
				inventory.NetworkID,
			)
		}
		checked++
	}

	if checked < 1000 {
		t.Fatalf("expected to verify at least 1000 mapped creative entries, verified %d", checked)
	}
	t.Logf("verified %d mapped creative entries against Pumpkin's normal inventory runtime IDs", checked)
}

func TestAllDamageableCreativeEquipmentUsesInventoryPaletteIdentity(t *testing.T) {
	l := &Listener{encoder: bedrockworld.NewEncoder()}
	l.initCreativeContent()

	advertised := make(map[uint32]protocol.ItemStack, len(l.creativeItems))
	for _, item := range l.creativeItems {
		advertised[item.CreativeItemNetworkID] = item.Item
	}

	wanted := make(map[string]struct{})
	for javaName := range javaToBedrockItemMappings {
		if player.MaxDurability(javaName) > 0 {
			wanted[javaName] = struct{}{}
		}
	}

	verified := make(map[string]struct{}, len(wanted))
	for creativeID, known := range l.creativeNames {
		if _, ok := wanted[known.name]; !ok {
			continue
		}

		creative, ok := advertised[creativeID]
		if !ok {
			t.Fatalf("creative equipment %d (%s) has no advertised stack", creativeID, known.name)
		}
		inventory := l.itemInstance(player.ItemStack{ItemID: known.name, Count: 1}, 1).Stack
		if creative.NetworkID != inventory.NetworkID || creative.MetadataValue != inventory.MetadataValue {
			t.Errorf(
				"creative equipment %d (%s) advertises palette identity %d/%d, normal inventory uses %d/%d",
				creativeID,
				known.name,
				creative.NetworkID,
				creative.MetadataValue,
				inventory.NetworkID,
				inventory.MetadataValue,
			)
		}
		if known.meta != 0 {
			t.Errorf("creative equipment %d (%s) starts with damage metadata %d", creativeID, known.name, known.meta)
		}
		if _, ok := creative.NBTData["Damage"]; ok {
			t.Errorf("creative equipment %d (%s) unexpectedly contains Damage NBT: %#v", creativeID, known.name, creative.NBTData)
		}
		verified[known.name] = struct{}{}
	}

	for javaName := range wanted {
		if _, ok := verified[javaName]; !ok {
			t.Errorf("damageable item %s is mapped but missing from the creative catalogue", javaName)
		}
	}
	if len(verified) != len(wanted) {
		t.Fatalf("verified %d of %d mapped damageable items", len(verified), len(wanted))
	}
	t.Logf("verified all %d mapped tools, weapons, armor pieces, and damageable utility items", len(verified))
}

func TestPumpkinCopperCreativeArmorIsPristineAndUnstackable(t *testing.T) {
	l := &Listener{encoder: bedrockworld.NewEncoder()}
	l.initCreativeContent()

	want := map[string]struct {
		durability int
		armor      int
	}{
		"minecraft:copper_helmet":     {durability: 121, armor: 2},
		"minecraft:copper_chestplate": {durability: 176, armor: 4},
		"minecraft:copper_leggings":   {durability: 165, armor: 3},
		"minecraft:copper_boots":      {durability: 143, armor: 1},
	}
	found := make(map[string]struct{}, len(want))
	for _, known := range l.creativeNames {
		expected, ok := want[known.name]
		if !ok {
			continue
		}
		if known.meta != 0 {
			t.Errorf("creative %s starts at damage %d", known.name, known.meta)
		}
		if got := player.MaxDurability(known.name); got != expected.durability {
			t.Errorf("creative %s durability = %d, want %d", known.name, got, expected.durability)
		}
		if got := player.ArmorPoints(known.name); got != expected.armor {
			t.Errorf("creative %s armor = %d, want %d", known.name, got, expected.armor)
		}
		if got := player.MaxStackSize(known.name); got != 1 {
			t.Errorf("creative %s max stack = %d, want 1", known.name, got)
		}
		found[known.name] = struct{}{}
	}
	for itemID := range want {
		if _, ok := found[itemID]; !ok {
			t.Errorf("Pumpkin creative catalogue is missing %s", itemID)
		}
	}
}

func TestEveryPumpkinCreativeEntryHasCurrentRuntimeID(t *testing.T) {
	for _, entry := range pumpkinCreativeItems {
		if _, ok := pumpkinCreativeCatalogueStack(entry); !ok {
			t.Errorf("creative entry %s/%d has no usable runtime ID", entry.name, entry.meta)
		}
	}
	if len(pumpkinCreativeItems) != 1875 {
		t.Fatalf("generated %d creative entries, want Pumpkin's complete 1875-entry catalogue", len(pumpkinCreativeItems))
	}
}

func TestBedrockOnlyCreativeRuntimeIDSurvivesInventorySync(t *testing.T) {
	l := &Listener{encoder: bedrockworld.NewEncoder()}
	l.initCreativeContent()

	for creativeID, known := range l.creativeNames {
		if known.name != "minecraft:normal_stone_slab" {
			continue
		}
		advertised := l.creativeItems[creativeID-1].Item
		inventory := l.itemInstance(player.ItemStack{ItemID: known.name, Count: 1, Damage: int(known.meta)}, 1).Stack
		if advertised.NetworkID != -899 || inventory.NetworkID != advertised.NetworkID {
			t.Fatalf("normal stone slab runtime ID advertised/inventory = %d/%d, want -899/-899", advertised.NetworkID, inventory.NetworkID)
		}
		return
	}
	t.Fatal("normal stone slab missing from Pumpkin creative catalogue")
}
