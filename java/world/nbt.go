package world

import (
	"encoding/binary"
	"io"
)

// NBT tag type constants used in the chunk heightmaps compound.
const (
	nbtTagEnd      byte = 0x00
	nbtTagLongArr  byte = 0x0C
	nbtTagCompound byte = 0x0A
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
	const (
		worldMinY      = -64
		bitsPerEntry   = 9
		entriesPerLong = 64 / bitsPerEntry // 7
		totalEntries   = 256               // 16×16 columns
		numLongs       = (totalEntries + entriesPerLong - 1) / entriesPerLong // 37
	)

	// value = Y of first non-solid block above the surface = surfaceY + 1
	// stored relative to WorldMinY
	value := int64(surfaceY+1) - int64(worldMinY)

	longs := make([]int64, numLongs)
	for i := range longs {
		var packed int64
		for j := 0; j < entriesPerLong; j++ {
			entryIdx := i*entriesPerLong + j
			if entryIdx < totalEntries {
				packed |= value << (j * bitsPerEntry)
			}
		}
		longs[i] = packed
	}
	return longs
}
