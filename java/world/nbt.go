package world

import (
	"bytes"
	"encoding/binary"
	"io"

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
)

// NBT tag type constants used in the chunk heightmaps compound.
const (
	nbtTagEnd      byte = 0x00
	nbtTagString   byte = 0x08
	nbtTagList     byte = 0x09
	nbtTagCompound byte = 0x0A
	nbtTagLongArr  byte = 0x0C
)

// writeNetworkNBTCompound writes an NBT compound in the "network NBT" format
// introduced in Minecraft 1.20.2.
//
// In network NBT the root compound has NO name (just the 0x0A type byte is
// written before the payload, without a preceding name length or name string).
// Each entry inside the compound is written normally: type + name + value.
// The compound is terminated by TAG_End (0x00).
//
// The caller provides fn to write the compound's entries; writeNetworkNBTCompound
// supplies the surrounding framing.
func writeNetworkNBTCompound(w io.Writer, fn func(io.Writer)) {
	w.Write([]byte{nbtTagCompound}) // root type, no name
	fn(w)
	w.Write([]byte{nbtTagEnd})
}

// writeNBTLongArray writes a TAG_Long_Array entry (type 0x0C) with the given
// name and values in big-endian byte order.
func writeNBTLongArray(w io.Writer, name string, longs []int64) {
	// Tag type
	w.Write([]byte{nbtTagLongArr})
	// Tag name: big-endian uint16 length + UTF-8 bytes
	nameBytes := []byte(name)
	var lenBuf [2]byte
	binary.BigEndian.PutUint16(lenBuf[:], uint16(len(nameBytes)))
	w.Write(lenBuf[:])
	w.Write(nameBytes)
	// Array length as big-endian int32
	var arrLenBuf [4]byte
	binary.BigEndian.PutUint32(arrLenBuf[:], uint32(len(longs)))
	w.Write(arrLenBuf[:])
	// Array values as big-endian int64
	var valBuf [8]byte
	for _, l := range longs {
		binary.BigEndian.PutUint64(valBuf[:], uint64(l))
		w.Write(valBuf[:])
	}
}

// packHeightmap packs 256 identical heightmap values into the compact long
// array used by the Java chunk format (1.18+, 9 bits per entry, no overflow).
//
// surfaceY is the absolute Y coordinate of the topmost solid block.
// The stored value is (surfaceY + 1) - WorldMinY, which for a surface at
// Y=63 with WorldMinY=-64 gives 128.
//
// Packing rules (Java 1.16+ "no overflow"):
//   - bitsPerEntry = 9 (max height = 384 < 512 = 2^9)
//   - entriesPerLong = floor(64 / 9) = 7
//   - longs needed   = ceil(256 / 7) = 37
//   - Entries are stored from the least-significant bit upward within each long.
//   - Entries do NOT span long boundaries (remaining high bits are zero).
func packHeightmap(surfaceY int) []int64 {
	var surfaceYs [256]int
	for i := range surfaceYs {
		surfaceYs[i] = surfaceY
	}
	return packHeightmapValues(surfaceYs)
}

// packHeightmapValues packs one absolute surface Y for each x/z column.
// Entries use index z*16+x, matching the chunk heightmap wire order.
func packHeightmapValues(surfaceYs [256]int) []int64 {
	const (
		worldMinY      = -64
		bitsPerEntry   = 9
		entriesPerLong = 64 / bitsPerEntry
		totalEntries   = 256
		numLongs       = (totalEntries + entriesPerLong - 1) / entriesPerLong
	)

	longs := make([]int64, numLongs)
	for entryIndex, surfaceY := range surfaceYs {
		value := int64(surfaceY+1) - int64(worldMinY)
		if value < 0 {
			value = 0
		} else if value > (1<<bitsPerEntry)-1 {
			value = (1 << bitsPerEntry) - 1
		}
		longIndex := entryIndex / entriesPerLong
		bitOffset := (entryIndex % entriesPerLong) * bitsPerEntry
		longs[longIndex] |= value << bitOffset
	}
	return longs
}

// BlockEntityNetworkData returns the network-NBT payload for a canonical block
// entity. Decorated-pot sherds are generated from canonical state so pots placed
// during this server session render correctly before the chunk is persisted.
func BlockEntityNetworkData(entity coreworld.BlockEntity) []byte {
	if entity.Type != "minecraft:decorated_pot" && entity.Type != "decorated_pot" {
		return entity.Data
	}
	decorations := player.NormalizePotDecorations(entity.PotDecorations)
	var buf bytes.Buffer
	writeNetworkNBTCompound(&buf, func(w io.Writer) {
		writeNBTStringList(w, "sherds", decorations[:])
	})
	return buf.Bytes()
}

func writeNBTStringList(w io.Writer, name string, values []string) {
	_, _ = w.Write([]byte{nbtTagList})
	writeNBTStringPayload(w, name)
	_, _ = w.Write([]byte{nbtTagString})
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(values)))
	_, _ = w.Write(length[:])
	for _, value := range values {
		writeNBTStringPayload(w, value)
	}
}

func writeNBTStringPayload(w io.Writer, value string) {
	data := []byte(value)
	var length [2]byte
	binary.BigEndian.PutUint16(length[:], uint16(len(data)))
	_, _ = w.Write(length[:])
	_, _ = w.Write(data)
}
