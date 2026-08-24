package world

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestTerrainBlockStateIDsProtocol769(t *testing.T) {
	tests := []struct {
		name string
		want int32
	}{
		{"stone", 1},
		{"grass_block", 9},
		{"dirt", 10},
		{"bedrock", 85},
		{"water", 86},
		{"sand", 118},
		{"gravel", 124},
		{"oak_log", 137},
		{"oak_leaves", 279},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			block := coreworld.Block{Namespace: "minecraft", Name: tc.name}
			if got := StateID(block); got != tc.want {
				t.Fatalf("StateID(minecraft:%s) = %d, want protocol-769 state %d", tc.name, got, tc.want)
			}
		})
	}
}

func TestProtocol769PlacementRegistryIDs(t *testing.T) {
	tests := []struct {
		name    string
		itemID  int32
		stateID int32
	}{
		{"acacia_planks", 40, 19},
		{"glass", 195, 562},
		{"crafting_table", 314, 4332},
		{"furnace", 316, 4350},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resource := "minecraft:" + tc.name
			if got := ItemID(resource); got != tc.itemID {
				t.Fatalf("ItemID(%q) = %d, want protocol-769 item ID %d", resource, got, tc.itemID)
			}
			if got := ItemName(tc.itemID); got != resource {
				t.Fatalf("ItemName(%d) = %q, want %q", tc.itemID, got, resource)
			}
			block := coreworld.Block{Namespace: "minecraft", Name: tc.name}
			if got := StateID(block); got != tc.stateID {
				t.Fatalf("StateID(%q) = %d, want protocol-769 default state %d", resource, got, tc.stateID)
			}
			if !IsPlaceableAsBlock(resource) {
				t.Fatalf("IsPlaceableAsBlock(%q) = false, want true", resource)
			}
		})
	}

	if got := len(itemIDs); got != 1385 {
		t.Fatalf("loaded item registry size = %d, want complete 1.21.4 count 1385", got)
	}
	if got := len(javaStateIDs); got < 27866 {
		t.Fatalf("loaded block-state registry size = %d, want all 27,866 protocol-769 states", got)
	}
	propertyStates := []struct {
		block coreworld.Block
		want  int32
	}{
		{coreworld.Block{Namespace: "minecraft", Name: "oak_log", Properties: map[string]string{"axis": "x"}}, 136},
		{coreworld.Block{Namespace: "minecraft", Name: "oak_stairs", Properties: map[string]string{"facing": "north", "half": "top", "shape": "straight", "waterlogged": "false"}}, 2930},
	}
	for _, test := range propertyStates {
		if got := StateID(test.block); got != test.want {
			t.Fatalf("StateID(%s)=%d, want %d", test.block.Key(), got, test.want)
		}
	}
}

func TestPackHeightmapValuesPreservesEveryColumn(t *testing.T) {
	var surfaces [256]int
	for i := range surfaces {
		surfaces[i] = coreworld.WorldMinY - 1 + i%190
	}
	packed := packHeightmapValues(surfaces)
	if len(packed) != 37 {
		t.Fatalf("packed long count = %d, want 37", len(packed))
	}
	const entriesPerLong = 7
	const mask = int64(0x1ff)
	for i, surface := range surfaces {
		stored := (packed[i/entriesPerLong] >> ((i % entriesPerLong) * 9)) & mask
		want := int64(surface + 1 - coreworld.WorldMinY)
		if stored != want {
			t.Fatalf("column %d stored height = %d, want %d (surface Y %d)", i, stored, want, surface)
		}
	}
}

func TestEncodeChunkHeightmapsUsesChunkColumns(t *testing.T) {
	chunk := &coreworld.Chunk{}
	for x := 0; x < coreworld.SectionSize; x++ {
		for z := 0; z < coreworld.SectionSize; z++ {
			y := 50 + x + z
			sectionIndex := (y - coreworld.WorldMinY) / coreworld.SectionSize
			localY := (y - coreworld.WorldMinY) % coreworld.SectionSize
			if chunk.Sections[sectionIndex] == nil {
				chunk.Sections[sectionIndex] = coreworld.NewSection()
			}
			chunk.Sections[sectionIndex].Set(x, localY, z, coreworld.Block{Namespace: "minecraft", Name: "stone"})
		}
	}

	// The NBT must differ from a constant-height map and be non-empty. Exact
	// per-column packing is independently asserted above.
	got := EncodeChunkHeightmaps(chunk)
	flat := EncodeHeightmaps(50)
	if len(got) == 0 {
		t.Fatal("dynamic heightmap encoding is empty")
	}
	if string(got) == string(flat) {
		t.Fatal("dynamic heightmap unexpectedly equals constant Y=50 map")
	}
}

func TestVillageBlockStatesExistInProtocol769Registry(t *testing.T) {
	states := []coreworld.Block{
		{Namespace: "minecraft", Name: "acacia_door", Properties: map[string]string{
			"facing": "south", "half": "lower", "hinge": "left",
			"open": "false", "powered": "false",
		}}, {Namespace: "minecraft", Name: "acacia_door", Properties: map[string]string{
			"facing": "south", "half": "upper", "hinge": "left",
			"open": "true", "powered": "false",
		}},
		{Namespace: "minecraft", Name: "farmland", Properties: map[string]string{"moisture": "7"}},
		{Namespace: "minecraft", Name: "wheat", Properties: map[string]string{"age": "7"}},
		{Namespace: "minecraft", Name: "acacia_stairs", Properties: map[string]string{
			"facing": "east", "half": "bottom", "shape": "straight", "waterlogged": "false",
		}},
		{Namespace: "minecraft", Name: "red_bed", Properties: map[string]string{
			"facing": "north", "occupied": "false", "part": "foot",
		}},
	}
	for _, state := range states {
		if !HasExactState(state) {
			t.Errorf("missing exact protocol-769 block state %s", state.Key())
		}
	}
}

func TestModernWorldgenBlocksAndBiomesExistInProtocol769Registry(t *testing.T) {
	blocks := []coreworld.Block{
		{Namespace: "minecraft", Name: "orange_terracotta"},
		{Namespace: "minecraft", Name: "yellow_terracotta"},
		{Namespace: "minecraft", Name: "brown_terracotta"},
		{Namespace: "minecraft", Name: "white_terracotta"},
		{Namespace: "minecraft", Name: "light_gray_terracotta"},
		{Namespace: "minecraft", Name: "coarse_dirt"},
		{Namespace: "minecraft", Name: "lava"},
		{Namespace: "minecraft", Name: "moss_block"},
		{Namespace: "minecraft", Name: "clay"},
		{Namespace: "minecraft", Name: "dripstone_block"},
		{Namespace: "minecraft", Name: "pointed_dripstone", Properties: map[string]string{"vertical_direction": "up", "thickness": "tip", "waterlogged": "false"}},
		{Namespace: "minecraft", Name: "sculk"},
		{Namespace: "minecraft", Name: "mycelium"},
		{Namespace: "minecraft", Name: "packed_ice"},
		{Namespace: "minecraft", Name: "blue_ice"},
		{Namespace: "minecraft", Name: "tube_coral_block"},
		{Namespace: "minecraft", Name: "mangrove_log"},
		{Namespace: "minecraft", Name: "pale_oak_log"},
		{Namespace: "minecraft", Name: "pale_oak_leaves"},
	}
	for _, block := range blocks {
		if StateID(block) == 0 {
			t.Errorf("worldgen block %s resolves to Java air", block.Key())
		}
		if len(block.Properties) != 0 && !HasExactState(block) {
			t.Errorf("missing exact protocol-769 worldgen state %s", block.Key())
		}
	}
	for _, biome := range coreworld.GeneratedBiomeNames() {
		name := "minecraft:" + biome
		if _, ok := biomeIDs[name]; !ok {
			t.Errorf("missing protocol-769 biome %s", name)
		}
	}
}

func TestCropStatesExistInProtocol769Registry(t *testing.T) {
	states := []coreworld.Block{
		{Namespace: "minecraft", Name: "wheat", Properties: map[string]string{"age": "0"}},
		{Namespace: "minecraft", Name: "wheat", Properties: map[string]string{"age": "7"}},
		{Namespace: "minecraft", Name: "carrots", Properties: map[string]string{"age": "7"}},
		{Namespace: "minecraft", Name: "potatoes", Properties: map[string]string{"age": "7"}},
		{Namespace: "minecraft", Name: "beetroots", Properties: map[string]string{"age": "3"}},
		{Namespace: "minecraft", Name: "nether_wart", Properties: map[string]string{"age": "3"}},
		{Namespace: "minecraft", Name: "torchflower_crop", Properties: map[string]string{"age": "1"}},
		{Namespace: "minecraft", Name: "pumpkin_stem", Properties: map[string]string{"age": "7"}},
		{Namespace: "minecraft", Name: "melon_stem", Properties: map[string]string{"age": "7"}},
		{Namespace: "minecraft", Name: "attached_pumpkin_stem", Properties: map[string]string{"facing": "east"}},
		{Namespace: "minecraft", Name: "attached_melon_stem", Properties: map[string]string{"facing": "west"}},
		{Namespace: "minecraft", Name: "sweet_berry_bush", Properties: map[string]string{"age": "3"}},
	}
	for _, state := range states {
		if !HasExactState(state) || StateID(state) == 0 {
			t.Errorf("missing exact Java crop state %s", state.Key())
		}
	}
}
