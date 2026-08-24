package world

import (
	"bytes"
	"encoding/binary"
	"testing"

	coreworld "GoCraft/core/world"
	dfworld "github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestCanonicalSubChunkUsesVersionNineAndAbsoluteY(t *testing.T) {
	section := coreworld.NewSection()
	section.Set(1, 2, 3, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	payload, err := NewEncoder().EncodeSubChunk(section, -3)
	if err != nil {
		t.Fatalf("EncodeSubChunk: %v", err)
	}
	if len(payload) < 4 {
		t.Fatalf("payload too short: %d", len(payload))
	}
	if payload[0] != 9 || payload[1] != 1 || int8(payload[2]) != -3 {
		t.Fatalf("header = [%d %d %d], want [9 1 -3]", payload[0], payload[1], int8(payload[2]))
	}
	if payload[3]&1 != 0 {
		t.Fatal("sub-chunk response palette is not marked as persistent")
	}
}

func TestCanonicalAirSectionUsesAllAirResult(t *testing.T) {
	payload, err := NewEncoder().EncodeSubChunk(nil, 0)
	if err != nil {
		t.Fatalf("EncodeSubChunk: %v", err)
	}
	if len(payload) != 0 {
		t.Fatalf("air payload length = %d, want 0", len(payload))
	}
}

func TestKnownBlocksResolveToDifferentNetworkHashes(t *testing.T) {
	encoder := NewEncoder()
	air := encoder.BlockNetworkID(coreworld.Air)
	stone := encoder.BlockNetworkID(coreworld.Block{Namespace: "minecraft", Name: "stone"})
	if air == stone {
		t.Fatalf("air and stone resolved to the same network ID %d", air)
	}
}

func TestModernWorldgenBlocksResolveForBedrock(t *testing.T) {
	encoder := NewEncoder()
	air := encoder.BlockNetworkID(coreworld.Air)
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
		if got := encoder.BlockNetworkID(block); got == air {
			t.Errorf("%s resolved to Bedrock air", block.Key())
		}
	}
}

func TestCropStagesAndAttachedStemsResolveForBedrock(t *testing.T) {
	encoder := NewEncoder()
	air := encoder.BlockNetworkID(coreworld.Air)
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
		{Namespace: "minecraft", Name: "bubble_column", Properties: map[string]string{"drag": "true"}},
		{Namespace: "minecraft", Name: "bubble_column", Properties: map[string]string{"drag": "false"}},
	}
	for _, state := range states {
		if got := encoder.BlockNetworkID(state); got == air {
			t.Errorf("crop state %s resolved to Bedrock air", state.Key())
		}
	}
	if young, mature := encoder.BlockNetworkID(states[0]), encoder.BlockNetworkID(states[1]); young == mature {
		t.Fatalf("young and mature wheat resolved to the same Bedrock state %d", young)
	}
	if down, up := encoder.BlockNetworkID(states[len(states)-2]), encoder.BlockNetworkID(states[len(states)-1]); down == up {
		t.Fatalf("upward and downward bubble columns resolved to the same Bedrock state %d", down)
	}
}

func TestFullChunkAirPaletteUsesNetworkHash(t *testing.T) {
	encoder := NewEncoder()
	payload, err := encoder.EncodeFullChunkPayload(nil)
	if err != nil {
		t.Fatalf("EncodeFullChunkPayload: %v", err)
	}
	if len(payload) < 5 || payload[0] != 9 || payload[3]&1 == 0 {
		t.Fatalf("first sub-chunk header = %v, want V9 network palette", payload[:min(len(payload), 4)])
	}
	var networkID int32
	if err := protocol.Varint32(bytes.NewBuffer(payload[4:]), &networkID); err != nil {
		t.Fatalf("decode air network hash: %v", err)
	}
	if got, want := uint32(networkID), encoder.BlockNetworkID(coreworld.Air); got != want {
		t.Fatalf("air network ID = %d, want stable hash %d", got, want)
	}
}

func TestBedrockBiomeMapCoversGeneratedOverworldBiomes(t *testing.T) {
	for _, biome := range coreworld.GeneratedBiomeNames() {
		name := "minecraft:" + biome
		if _, ok := javaToBedrockBiomeID[name]; !ok {
			t.Errorf("no Bedrock runtime biome mapping for %s", name)
		}
	}
}

func TestBiomeStorageExpandsQuartCellsForBedrock(t *testing.T) {
	section := coreworld.NewSection()
	section.SetUniformBiome("minecraft:plains")
	section.SetBiomeCell(1, 2, 3, "minecraft:lush_caves")
	payload := NewEncoder().encodeBiomeStorage(section)
	if bits := payload[0] >> 1; bits != 1 {
		t.Fatalf("biome bits per entry = %d, want 1", bits)
	}

	// Quart cell (1,2,3) expands to block coordinates x=4..7, y=8..11,
	// z=12..15. Bedrock's linear order is X-Z-Y.
	inside := 5*256 + 13*16 + 9
	outside := 1*256 + 1*16 + 1
	if got := packedPaletteIndex(payload[1:], inside, 1); got != 1 {
		t.Fatalf("palette index inside cave quart = %d, want 1", got)
	}
	if got := packedPaletteIndex(payload[1:], outside, 1); got != 0 {
		t.Fatalf("palette index outside cave quart = %d, want 0", got)
	}

	paletteOffset := 1 + 128*4
	buf := bytes.NewBuffer(payload[paletteOffset:])
	var count, plainsID, lushID int32
	if err := protocol.Varint32(buf, &count); err != nil {
		t.Fatalf("decode biome palette count: %v", err)
	}
	if err := protocol.Varint32(buf, &plainsID); err != nil {
		t.Fatalf("decode plains biome ID: %v", err)
	}
	if err := protocol.Varint32(buf, &lushID); err != nil {
		t.Fatalf("decode lush cave biome ID: %v", err)
	}
	if count != 2 || plainsID != 1 || lushID != 187 {
		t.Fatalf("biome palette = count %d IDs [%d %d], want count 2 IDs [1 187]", count, plainsID, lushID)
	}
}

func TestCanonicalSubChunkReordersBlocksForBedrock(t *testing.T) {
	section := coreworld.NewSection()
	section.Set(1, 2, 3, coreworld.Block{Namespace: "minecraft", Name: "stone"})

	payload, err := NewEncoder().EncodeSubChunk(section, 0)
	if err != nil {
		t.Fatalf("EncodeSubChunk: %v", err)
	}
	if bits := payload[3] >> 1; bits != 1 {
		t.Fatalf("bits per block = %d, want 1", bits)
	}

	// Bedrock's flat index is X*256 + Z*16 + Y. The canonical section's
	// Y*256 + Z*16 + X index must not leak directly onto the wire.
	bedrockIndex := 1*256 + 3*16 + 2
	canonicalIndex := 2*256 + 3*16 + 1
	if got := packedPaletteIndex(payload[4:], bedrockIndex, 1); got != 1 {
		t.Fatalf("palette index at Bedrock position = %d, want 1", got)
	}
	if got := packedPaletteIndex(payload[4:], canonicalIndex, 1); got != 0 {
		t.Fatalf("palette index at unconverted canonical position = %d, want 0", got)
	}
}

func packedPaletteIndex(words []byte, index, bits int) uint32 {
	valuesPerWord := 32 / bits
	wordIndex := index / valuesPerWord
	bitOffset := (index % valuesPerWord) * bits
	word := binary.LittleEndian.Uint32(words[wordIndex*4:])
	return (word >> bitOffset) & (1<<bits - 1)
}

func TestBlockNetworkIDsUseCurrentPumpkinPaletteHashes(t *testing.T) {
	encoder := NewEncoder()
	air := encoder.BlockNetworkID(coreworld.Air)
	stone := encoder.BlockNetworkID(coreworld.Block{Namespace: `minecraft`, Name: `stone`})
	if len(encoder.byName[`minecraft:air`]) == 0 || len(encoder.byName[`minecraft:stone`]) == 0 {
		t.Fatal("embedded Pumpkin palette is missing air or stone")
	}
	if air == 0 || stone == 0 || air == stone {
		t.Fatalf(`palette hashes are invalid: air=%d stone=%d`, air, stone)
	}
	// Network hashes are stable: common states must also agree with the
	// independently embedded Dragonfly registry used elsewhere in the adapter.
	registry := dfworld.DefaultBlockRegistry
	registry.Finalize()
	for name, got := range map[string]uint32{"minecraft:air": air, "minecraft:stone": stone} {
		matched := false
		for runtimeID := uint32(0); runtimeID < uint32(registry.BlockCount()); runtimeID++ {
			candidate, _, ok := registry.RuntimeIDToState(runtimeID)
			if !ok || candidate != name {
				continue
			}
			want, ok := registry.RuntimeIDToHash(runtimeID)
			if ok && want == got {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("Pumpkin %s hash %d does not match the independent registry", name, got)
		}
	}
}
