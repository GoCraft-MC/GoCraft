// Package world provides Bedrock Edition world encoding utilities for GoCraft.
//
// M14.1 scope: minimal flat-world sub-chunk encoding to get a Bedrock client
// into a walkable world.  Full palette-driven block encoding (matching
// core/world blocks to Bedrock runtime IDs) is deferred to M14.2.
package world

import (
	"bytes"
	"encoding/binary"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

// Overworld sub-chunk layout for Bedrock 1.18+ (Y range −64..320):
//
//	Sub-chunk 0 → Y = −64 to −49
//	Sub-chunk 8 → Y =   64 to  79   ← ground level for flat world
//	Sub-chunk 23 → Y = 304 to 319
const (
	groundSubChunkIndex = int32(8) // contains Y=64..79
	spawnY              = float32(65)
)

// SpawnY returns the Y coordinate players should spawn at in the flat world.
func SpawnY() float32 { return spawnY }

// GroundSubChunkIndex returns the sub-chunk index that contains the surface.
func GroundSubChunkIndex() int32 { return groundSubChunkIndex }

// blockState is the NBT structure of a Bedrock block palette entry.
type blockState struct {
	Name   string         `nbt:"name"`
	States map[string]any `nbt:"states"`
}

// EncodeAirSubChunk returns a valid version-8 sub-chunk payload that contains
// only air.  Uses a single-value (bitsPerBlock=0) persistent palette to
// minimise payload size.
//
// Format:
//
//	0x08           version 8
//	0x01           bitsPerBlock=0 | persistentFlag=1  → (0<<1)|1 = 1
//	LE int32 = 1   palette count
//	NBT compound   minecraft:air
func EncodeAirSubChunk() ([]byte, error) {
	airNBT, err := marshalBlockState("minecraft:air", nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteByte(0x08) // version
	buf.WriteByte(0x01) // bitsPerBlock=0, persistent
	writeInt32LE(&buf, 1)
	buf.Write(airNBT)
	return buf.Bytes(), nil
}

// EncodeGroundSubChunk returns a valid version-8 sub-chunk payload for the
// ground level (sub-chunk index 8, Y=64..79 in the overworld).
//
// Block layout:
//   - Y=0 (global Y=64): all 256 blocks = minecraft:stone
//   - Y=1..15 (global Y=65..79): all blocks = air
//
// Format:
//
//	0x08                version 8
//	0x03                bitsPerBlock=1 | persistentFlag=1 → (1<<1)|1 = 3
//	128 LE uint32s      packed block indices (1 bit each, 32 per word)
//	  [8 × 0xFFFFFFFF] Y=0 blocks (stone, palette index 1)
//	  [120 × 0x00000000] Y=1..15 (air, palette index 0)
//	LE int32 = 2        palette count
//	NBT compound        minecraft:air   (index 0)
//	NBT compound        minecraft:stone (index 1)
func EncodeGroundSubChunk() ([]byte, error) {
	airNBT, err := marshalBlockState("minecraft:air", nil)
	if err != nil {
		return nil, err
	}
	stoneNBT, err := marshalBlockState("minecraft:stone", nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteByte(0x08) // version 8
	buf.WriteByte(0x03) // bitsPerBlock=1, persistent

	// 128 × LE uint32 (1 bit per block, 4096 blocks total, 32 per uint32).
	// Block index order within a sub-chunk: idx = x | (z << 4) | (y << 8)
	// Y=0: indices 0..255 → first 8 words → all ones (palette index 1 = stone)
	// Y=1..15: indices 256..4095 → remaining 120 words → all zeros (air)
	word := make([]byte, 4)
	for i := range 128 {
		var w uint32
		if i < 8 {
			w = 0xFFFF_FFFF // Y=0 layer: stone
		}
		binary.LittleEndian.PutUint32(word, w)
		buf.Write(word)
	}

	writeInt32LE(&buf, 2) // 2 palette entries
	buf.Write(airNBT)
	buf.Write(stoneNBT)
	return buf.Bytes(), nil
}

// marshalBlockState serialises a Bedrock block state to Network Little Endian
// NBT format, as required by the persistent sub-chunk palette.
// states may be nil for blocks with no properties.
func marshalBlockState(name string, states map[string]any) ([]byte, error) {
	if states == nil {
		states = map[string]any{}
	}
	return nbt.MarshalEncoding(blockState{Name: name, States: states}, nbt.NetworkLittleEndian)
}

func writeInt32LE(buf *bytes.Buffer, v int32) {
	b := [4]byte{}
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	buf.Write(b[:])
}
