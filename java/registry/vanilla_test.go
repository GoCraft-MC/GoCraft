package registry

import (
	"reflect"
	"testing"

	"GoCraft/java/protocol"
)

func TestSynchronizedRegistryData769(t *testing.T) {
	wantNames := []string{
		"minecraft:worldgen/biome",
		"minecraft:chat_type",
		"minecraft:trim_pattern",
		"minecraft:trim_material",
		"minecraft:wolf_variant",
		"minecraft:painting_variant",
		"minecraft:dimension_type",
		"minecraft:damage_type",
		"minecraft:banner_pattern",
		"minecraft:enchantment",
		"minecraft:jukebox_song",
		"minecraft:instrument",
	}
	wantCounts := []int{65, 7, 18, 11, 9, 50, 4, 49, 43, 42, 19, 8}

	if !reflect.DeepEqual(synchronizedRegistryOrder769, wantNames) {
		t.Fatalf("protocol-769 synchronized registry order\n got: %v\nwant: %v", synchronizedRegistryOrder769, wantNames)
	}
	if len(vanillaNetworkRegistries) != len(wantNames) {
		t.Fatalf("synchronized registry count = %d, want %d", len(vanillaNetworkRegistries), len(wantNames))
	}

	provider := &VanillaProvider{}
	dimensionID, err := provider.DimensionTypeID("minecraft:overworld")
	if err != nil {
		t.Fatal(err)
	}

	for registryIndex, registry := range vanillaNetworkRegistries {
		if registry.Name != wantNames[registryIndex] {
			t.Fatalf("registry %d name = %q, want %q", registryIndex, registry.Name, wantNames[registryIndex])
		}
		if len(registry.Entries) != wantCounts[registryIndex] {
			t.Fatalf("registry %s entry count = %d, want %d", registry.Name, len(registry.Entries), wantCounts[registryIndex])
		}

		pkt := buildRegistryDataPacket(registry)
		if pkt.ID != 0x07 {
			t.Fatalf("Registry Data packet ID = 0x%02x, want 0x07", pkt.ID)
		}

		r := pkt.Reader()
		registryName, err := protocol.ReadString(r)
		if err != nil {
			t.Fatalf("registry %d name: %v", registryIndex, err)
		}
		if registryName != wantNames[registryIndex] {
			t.Fatalf("decoded registry %d name = %q, want %q", registryIndex, registryName, wantNames[registryIndex])
		}
		count, err := protocol.ReadVarInt(r)
		if err != nil {
			t.Fatalf("registry %s count: %v", registryName, err)
		}
		if count != int32(wantCounts[registryIndex]) {
			t.Fatalf("decoded registry %s count = %d, want %d", registryName, count, wantCounts[registryIndex])
		}

		for registryID := int32(0); registryID < count; registryID++ {
			name, err := protocol.ReadString(r)
			if err != nil {
				t.Fatalf("registry %s entry %d name: %v", registryName, registryID, err)
			}
			if name != registry.Entries[registryID] {
				t.Fatalf("registry %s entry %d = %q, want %q", registryName, registryID, name, registry.Entries[registryID])
			}
			hasData, err := protocol.ReadBool(r)
			if err != nil {
				t.Fatalf("registry %s entry %d data presence: %v", registryName, registryID, err)
			}
			if hasData {
				t.Fatalf("registry %s entry %d %q unexpectedly contains NBT despite selected known pack", registryName, registryID, name)
			}
			if registryName == dimensionTypeRegistryName && name == "minecraft:overworld" && registryID != dimensionID {
				t.Fatalf("Play Login overworld ID = %d, but Configuration assigned %d", dimensionID, registryID)
			}
		}
		if r.Len() != 0 {
			t.Fatalf("registry %s packet has %d trailing bytes", registryName, r.Len())
		}
	}

	if dimensionID != 0 {
		t.Fatalf("overworld dimension type ID = %d, want 0", dimensionID)
	}
}

func TestUpdateTags769IndependentDecode(t *testing.T) {
	wantNames := []string{
		"minecraft:worldgen/biome",
		"minecraft:painting_variant",
		"minecraft:damage_type",
		"minecraft:banner_pattern",
		"minecraft:enchantment",
		"minecraft:instrument",
		"minecraft:block",
		"minecraft:cat_variant",
		"minecraft:entity_type",
		"minecraft:fluid",
		"minecraft:game_event",
		"minecraft:item",
		"minecraft:point_of_interest_type",
	}
	wantRegistrySizes := []int32{65, 50, 49, 43, 42, 8, 1095, 11, 149, 5, 60, 1385, 20}
	wantTagCounts := []int32{70, 1, 33, 11, 22, 3, 186, 2, 35, 2, 5, 173, 3}

	if !reflect.DeepEqual(networkTagRegistryOrder769, wantNames) {
		t.Fatalf("protocol-769 network tag registry order\n got: %v\nwant: %v", networkTagRegistryOrder769, wantNames)
	}
	pkt := buildUpdateTagsPacket(vanillaNetworkTags)
	if pkt.ID != 0x0d {
		t.Fatalf("Update Tags packet ID = 0x%02x, want 0x0d", pkt.ID)
	}

	r := pkt.Reader()
	registryCount, err := protocol.ReadVarInt(r)
	if err != nil {
		t.Fatalf("registry count: %v", err)
	}
	if registryCount != int32(len(wantNames)) {
		t.Fatalf("registry count = %d, want %d", registryCount, len(wantNames))
	}

	var totalTags int32
	var totalIDs int
	for registryIndex := int32(0); registryIndex < registryCount; registryIndex++ {
		registryName, err := protocol.ReadString(r)
		if err != nil {
			t.Fatalf("registry %d name: %v", registryIndex, err)
		}
		if registryName != wantNames[registryIndex] {
			t.Fatalf("registry %d name = %q, want %q", registryIndex, registryName, wantNames[registryIndex])
		}
		if vanillaNetworkTags[registryIndex].RegistrySize != wantRegistrySizes[registryIndex] {
			t.Fatalf("registry %s size = %d, want %d", registryName, vanillaNetworkTags[registryIndex].RegistrySize, wantRegistrySizes[registryIndex])
		}

		tagCount, err := protocol.ReadVarInt(r)
		if err != nil {
			t.Fatalf("registry %s tag count: %v", registryName, err)
		}
		if tagCount != wantTagCounts[registryIndex] {
			t.Fatalf("registry %s tag count = %d, want %d", registryName, tagCount, wantTagCounts[registryIndex])
		}
		totalTags += tagCount

		for tagIndex := int32(0); tagIndex < tagCount; tagIndex++ {
			tagName, err := protocol.ReadString(r)
			if err != nil {
				t.Fatalf("registry %s tag %d name: %v", registryName, tagIndex, err)
			}
			wantTag := vanillaNetworkTags[registryIndex].Tags[tagIndex]
			if tagName != wantTag.Name {
				t.Fatalf("registry %s tag %d = %q, want %q", registryName, tagIndex, tagName, wantTag.Name)
			}
			entryCount, err := protocol.ReadVarInt(r)
			if err != nil {
				t.Fatalf("registry %s tag %s entry count: %v", registryName, tagName, err)
			}
			if entryCount != int32(len(wantTag.Entries)) {
				t.Fatalf("registry %s tag %s entry count = %d, want %d", registryName, tagName, entryCount, len(wantTag.Entries))
			}
			for entryIndex := int32(0); entryIndex < entryCount; entryIndex++ {
				id, err := protocol.ReadVarInt(r)
				if err != nil {
					t.Fatalf("registry %s tag %s entry %d: %v", registryName, tagName, entryIndex, err)
				}
				if id != wantTag.Entries[entryIndex] {
					t.Fatalf("registry %s tag %s entry %d = %d, want %d", registryName, tagName, entryIndex, id, wantTag.Entries[entryIndex])
				}
				if id < 0 || id >= wantRegistrySizes[registryIndex] {
					t.Fatalf("registry %s tag %s contains out-of-range ID %d", registryName, tagName, id)
				}
				totalIDs++
			}
		}
	}
	if totalTags != 546 {
		t.Fatalf("total tags = %d, want 546", totalTags)
	}
	if totalIDs != 6609 {
		t.Fatalf("total tag IDs = %d, want 6609", totalIDs)
	}
	if r.Len() != 0 {
		t.Fatalf("Update Tags packet has %d trailing bytes", r.Len())
	}
}

func TestDimensionTypeRegistryOrder769(t *testing.T) {
	want := []string{
		"minecraft:overworld",
		"minecraft:overworld_caves",
		"minecraft:the_end",
		"minecraft:the_nether",
	}
	for _, registry := range vanillaNetworkRegistries {
		if registry.Name == dimensionTypeRegistryName {
			if !reflect.DeepEqual(registry.Entries, want) {
				t.Fatalf("dimension_type entries\n got: %v\nwant: %v", registry.Entries, want)
			}
			return
		}
	}
	t.Fatalf("%s not found", dimensionTypeRegistryName)
}
