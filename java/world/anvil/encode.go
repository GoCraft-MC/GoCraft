package anvil

import (
	"bytes"
	"math/bits"

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

	// biomes TAG_Compound (minimal — single biome for the whole section)
	writeCompoundOpen(buf, "biomes")
	writeBiomes(buf, sec.Biome)
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
		// Properties TAG_Compound: each property is a named TAG_String.
		writeCompoundOpen(buf, "Properties")
		for k, v := range blk.Properties {
			writeTagString(buf, k, v)
		}
		writeCompoundEnd(buf)
	}

	// Close palette entry compound.
	writeCompoundEnd(buf)
}

// writeBiomes writes a minimal biomes section: a single-entry palette mapping
// the whole section to one biome resource location.
func writeBiomes(buf *bytes.Buffer, biome string) {
	if biome == "" {
		biome = "minecraft:plains"
	}
	writeListHeader(buf, "palette", tagString, 1)
	writeMUTF8(buf, biome)
	// No "data" array needed for a single-entry palette.
}
