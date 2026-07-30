package world

import (
	"bytes"
	"encoding/binary"
	"io"

	coreworld "GoCraft/core/world"
)

// EncodeHeightmaps returns the raw bytes of the NBT compound that goes into
// the Heightmaps field of the Level Chunk With Light packet (1.21.4).
//
// Two heightmap arrays are always included:
//   - MOTION_BLOCKING: the topmost block that prevents entity movement.
//   - WORLD_SURFACE:   the topmost non-air block (same for a flat world).
//
// Both arrays are packed long arrays with 9 bits per entry and 37 longs,
// encoding 256 identical values (one per column in the 16×16 chunk).
func EncodeHeightmaps(surfaceY int) []byte {
	longs := packHeightmap(surfaceY)
	var buf bytes.Buffer
	writeNetworkNBTCompound(&buf, func(w io.Writer) {
		writeNBTLongArray(w, "MOTION_BLOCKING", longs)
		writeNBTLongArray(w, "WORLD_SURFACE", longs)
	})
	return buf.Bytes()
}

// EncodeChunk converts a canonical Chunk into the raw byte slice that is sent
// as the Data (VarInt-prefixed ByteArray) field of Level Chunk With Light.
//
// Layout: for each of the 24 sections, bottom-to-top:
//
//	Int16              non-air block count
//	PalettedContainer  block states (indirect palette, ≥4 bits, or single value)
//	PalettedContainer  biomes       (single value for Milestone 4)
func EncodeChunk(c *coreworld.Chunk) []byte {
	var buf bytes.Buffer
	for _, sec := range c.Sections {
		encodeSection(&buf, sec)
	}
	return buf.Bytes()
}

// encodeSection writes one 16×16×16 section into buf.
func encodeSection(buf *bytes.Buffer, sec *coreworld.Section) {
	if sec == nil || sec.NonAir == 0 {
		// All-air section: block count = 0, single-value air, single-value biome.
		writeInt16BE(buf, 0)
		writeSingleValue(buf, 0) // air state ID = 0
		writeSingleValue(buf, 1) // biome: plains = 1 (vanilla registry, approximate)
		return
	}

	// Non-trivial section.
	writeInt16BE(buf, sec.NonAir)
	writeBlockStates(buf, sec)
	writeSingleValue(buf, javabiomeID(sec.Biome)) // per-section biome (M4 simplification)
}

// writeSingleValue encodes a single-value PalettedContainer:
//
//	Byte(0)        bits_per_entry = 0 → single value
//	VarInt(value)  the global state / biome ID
//	VarInt(0)      data array length = 0
func writeSingleValue(buf *bytes.Buffer, value int32) {
	buf.WriteByte(0)
	writeVarInt(buf, value)
	writeVarInt(buf, 0)
}

// writeBlockStates encodes a Section's blocks as an indirect PalettedContainer.
//
// Java requires at least 4 bits per entry for block states (indirect palette
// threshold is 9; direct palette uses exactly 15 bits per entry).
// We always use indirect for palettes ≤ 256 entries.
//
// Wire layout (indirect):
//
//	Byte         bits_per_entry (≥ 4)
//	VarInt       palette size
//	VarInt[]     palette entries (Java global state IDs)
//	VarInt       data array length (in longs)
//	Long[]       packed bit array (no overflow: entries stay within one long)
func writeBlockStates(buf *bytes.Buffer, sec *coreworld.Section) {
	palette := sec.BlockPalette()

	// Map each canonical Block to its Java 1.21.4 global state ID.
	// The StateID lookup is purely a Java-adapter concern; the core palette
	// holds canonical Block values with no knowledge of Java IDs.
	javaIDs := make([]int32, len(palette))
	for i, block := range palette {
		javaIDs[i] = StateID(block)
	}

	// Determine bits per entry: minimum 4 for Java block states.
	bitsPerEntry := bitsNeeded(len(javaIDs))
	if bitsPerEntry < 4 {
		bitsPerEntry = 4
	}
	// If bitsPerEntry ≥ 9 we would need the direct (global) format; for M4
	// palettes are tiny so this branch is unreachable.

	buf.WriteByte(byte(bitsPerEntry))

	// Palette
	writeVarInt(buf, int32(len(javaIDs)))
	for _, id := range javaIDs {
		writeVarInt(buf, id)
	}

	// Pack block data: no-overflow packing (Java 1.16+ format).
	entriesPerLong := 64 / bitsPerEntry
	dataLen := (4096 + entriesPerLong - 1) / entriesPerLong // ceil(4096 / epl)
	writeVarInt(buf, int32(dataLen))

	data := sec.BlockData()
	var longBuf [8]byte
	for longIdx := 0; longIdx < dataLen; longIdx++ {
		var packed int64
		base := longIdx * entriesPerLong
		for entry := 0; entry < entriesPerLong; entry++ {
			blockIdx := base + entry
			if blockIdx >= 4096 {
				break
			}
			packed |= int64(data[blockIdx]) << (entry * bitsPerEntry)
		}
		binary.BigEndian.PutUint64(longBuf[:], uint64(packed))
		buf.Write(longBuf[:])
	}
}

// javabiomeID returns the vanilla 1.21.4 biome registry ID for a canonical
// biome name.  Unknown names fall back to 1 (minecraft:plains, approximate).
//
// Milestone 13 will replace this with a registry lookup.
func javabiomeID(name string) int32 {
	switch name {
	case "minecraft:plains":
		return 1
	case "minecraft:the_void":
		return 0
	default:
		return 1 // default to plains
	}
}

// bitsNeeded returns ceil(log2(n)), minimum 1.
func bitsNeeded(n int) int {
	if n <= 1 {
		return 1
	}
	bits := 0
	n--
	for n > 0 {
		n >>= 1
		bits++
	}
	return bits
}

// writeInt16BE writes a big-endian signed 16-bit integer.
func writeInt16BE(buf *bytes.Buffer, v int16) {
	buf.Write([]byte{byte(v >> 8), byte(v)})
}

// writeVarInt appends a VarInt to buf.
func writeVarInt(buf *bytes.Buffer, v int32) {
	uv := uint32(v)
	for {
		if uv < 0x80 {
			buf.WriteByte(byte(uv))
			return
		}
		buf.WriteByte(byte(uv&0x7F) | 0x80)
		uv >>= 7
	}
}
