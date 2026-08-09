package bedrock

import (
	"strconv"
	"testing"

	bedrockworld "GoCraft/bedrock/world"
	"GoCraft/core/intent"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"

	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

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

func TestOnlyUnsupportedPumpkinCreativeEntriesAreSkipped(t *testing.T) {
	dfworld.DefaultBlockRegistry.Finalize()
	var root struct {
		Groups []creativeNBTGroup `nbt:"groups"`
		Items  []creativeNBTItem  `nbt:"items"`
	}
	if err := nbt.Unmarshal(creativeItemsNBT, &root); err != nil {
		t.Fatal(err)
	}
	expected := map[string]struct{}{
		"minecraft:sulfur_wall":            {},
		"minecraft:polished_sulfur_wall":   {},
		"minecraft:sulfur_brick_wall":      {},
		"minecraft:sulfur_stairs":          {},
		"minecraft:polished_sulfur_stairs": {},
		"minecraft:sulfur_brick_stairs":    {},
		"minecraft:sulfur_slab":            {},
		"minecraft:polished_sulfur_slab":   {},
		"minecraft:sulfur_brick_slab":      {},
		"minecraft:sulfur_bricks":          {},
		"minecraft:chiseled_sulfur":        {},
		"minecraft:sulfur":                 {},
		"minecraft:polished_sulfur":        {},
		"minecraft:potent_sulfur":          {},
		"minecraft:sulfur_spike":           {},
		"minecraft:sulfur_cube_spawn_egg":  {},
		"minecraft:music_disc_bounce":      {},
		"minecraft:sulfur_cube_bucket":     {},
	}
	seen := make(map[string]struct{}, len(expected))
	for _, entry := range root.Items {
		if _, ok := creativeItemStack(entry); !ok {
			if _, allowed := expected[entry.Name]; !allowed {
				t.Errorf("unexpectedly skipped creative entry %s/%d", entry.Name, entry.Meta)
			}
			seen[entry.Name] = struct{}{}
		}
	}
	for name := range expected {
		if _, ok := seen[name]; !ok {
			t.Errorf("expected unsupported entry %s was not present in Pumpkin's creative source", name)
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("skipped %d unique creative entries, want exactly %d known unsupported entries", len(seen), len(expected))
	}
}
