package anvil

import (
	"bytes"
	"math/bits"
	"sort"

	coreworld "GoCraft/core/world"
)

// dataVersion is written into every saved chunk for tool compatibility.
// Value 4189 corresponds to Minecraft Java Edition 1.21.4.
// Estimate; verify against Minecraft's generated-data output if external tools
// need to open the world.
const dataVersion = int32(4189)

// encodeChunkNBT serialises a canonical core/world.Chunk to uncompressed Anvil
// chunk NBT (standard file-format encoding with root compound name "").
//
// The caller is responsible for compressing the returned bytes (zlib, type 2).
func encodeChunkNBT(c *coreworld.Chunk) []byte {
	var buf bytes.Buffer

	// Root TAG_Compound header: type byte (0x0A) + name (empty).
	wByte(&buf, byte(tagCompound))
	writeMUTF8(&buf, "")

	writeTagInt(&buf, "DataVersion", dataVersion)
	writeTagInt(&buf, "xPos", c.X)
	writeTagInt(&buf, "yPos", int32(sectionYBase)) // minimum section Y (-4)
	writeTagInt(&buf, "zPos", c.Z)
	writeTagString(&buf, "Status", "minecraft:full")

	// sections TAG_List — count non-nil sections first.
	var nonNil []int
	for i, sec := range c.Sections {
		if sec != nil {
			nonNil = append(nonNil, i)
		}
	}

	writeListHeader(&buf, "sections", tagCompound, len(nonNil))
	for _, sIdx := range nonNil {
		sec := c.Sections[sIdx]
		encodeSectionPayload(&buf, sIdx, sec)
	}

	// Heightmaps are required for tool compatibility and are recomputed from
	// the canonical block columns rather than copied from a generator preset.
	packedHeightmap := packChunkHeightmap(c)
	writeCompoundOpen(&buf, "Heightmaps")
	writeTagLongArray(&buf, "MOTION_BLOCKING", packedHeightmap)
	writeTagLongArray(&buf, "WORLD_SURFACE", packedHeightmap)
	writeCompoundEnd(&buf)

	// Persist the complete opaque block-entity payload alongside its canonical
	// Java type and absolute position.
	writeNamedHeader(&buf, tagList, "block_entities")
	writePayload(&buf, blockEntitiesTag(c.BlockEntities))

	// Close root compound.
	writeCompoundEnd(&buf)

	return buf.Bytes()
}

// encodeSectionPayload writes the TAG_Compound payload for one section.
// (No type byte or name — it is a list element; the list header already
// specified the element type.)
func encodeSectionPayload(buf *bytes.Buffer, sIdx int, sec *coreworld.Section) {
	// Section Y coordinate.
	sY := int8(sIdx + sectionYBase)

	// Write the compound contents (no header — we are inside a list).
	// TAG_Byte "Y"
	writeTagByte(buf, "Y", sY)

	// block_states TAG_Compound
	writeCompoundOpen(buf, "block_states")
	encodeBlockStates(buf, sec)
	writeCompoundEnd(buf)

	// biomes TAG_Compound (full 4x4x4 paletted container)
	writeCompoundOpen(buf, "biomes")
	writeBiomes(buf, sec)
	writeCompoundEnd(buf)

	// Close the section compound.
	writeCompoundEnd(buf)
}

// encodeBlockStates writes the "palette" list and optional "data" long array
// inside an already-open block_states TAG_Compound.
func encodeBlockStates(buf *bytes.Buffer, sec *coreworld.Section) {
	palette := sec.BlockPalette()
	data := sec.BlockData()

	// palette TAG_List (TAG_Compound entries)
	writeListHeader(buf, "palette", tagCompound, len(palette))
	for _, blk := range palette {
		encodePaletteEntry(buf, blk)
	}

	// If palette has more than one entry, emit the packed long array.
	if len(palette) <= 1 {
		return
	}

	bitsPerEntry := max(4, bits.Len(uint(len(palette)-1)))
	entriesPerLong := 64 / bitsPerEntry
	numLongs := (4096 + entriesPerLong - 1) / entriesPerLong

	longs := make([]int64, numLongs)
	for i, v := range data {
		longIdx := i / entriesPerLong
		bitOff := (i % entriesPerLong) * bitsPerEntry
		longs[longIdx] |= int64(v) << bitOff
	}

	writeTagLongArray(buf, "data", longs)
}

// encodePaletteEntry writes one TAG_Compound block-state palette entry.
// (No header — it is inside a TAG_List; the element type is already stated.)
func encodePaletteEntry(buf *bytes.Buffer, blk coreworld.Block) {
	// TAG_String "Name"
	writeTagString(buf, "Name", blk.ResourceLocation())

	if len(blk.Properties) > 0 {
		// Sort property names for deterministic fixtures and region diffs.
		writeCompoundOpen(buf, "Properties")
		keys := make([]string, 0, len(blk.Properties))
		for key := range blk.Properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			writeTagString(buf, key, blk.Properties[key])
		}
		writeCompoundEnd(buf)
	}

	// Close palette entry compound.
	writeCompoundEnd(buf)
}

// writeBiomes serialises the section's 4x4x4 biome paletted container.
func writeBiomes(buf *bytes.Buffer, section *coreworld.Section) {
	palette := section.BiomePalette()
	data := section.BiomeData()
	writeListHeader(buf, "palette", tagString, len(palette))
	for _, biome := range palette {
		writeMUTF8(buf, biome)
	}
	if len(palette) <= 1 {
		return
	}
	bitsPerEntry := max(1, bits.Len(uint(len(palette)-1)))
	entriesPerLong := 64 / bitsPerEntry
	numLongs := (64 + entriesPerLong - 1) / entriesPerLong
	longs := make([]int64, numLongs)
	for cell, value := range data {
		longIndex := cell / entriesPerLong
		bitOffset := (cell % entriesPerLong) * bitsPerEntry
		longs[longIndex] |= int64(value) << bitOffset
	}
	writeTagLongArray(buf, "data", longs)
}
