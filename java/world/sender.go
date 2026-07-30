package world

import (
	"fmt"

	coreworld "GoCraft/core/world"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

// packetIDLevelChunkWithLight is the S→C Play packet ID for
// "Level Chunk With Light" in Minecraft Java Edition 1.21.4 (protocol 769).
//
// The ID may need adjustment after live testing against the vanilla client.
// Check wiki.vg/Protocol when updating the server version.
const packetIDLevelChunkWithLight = 0x27

// Sender converts canonical Chunks into Java chunk packets and writes them
// to a ClientConn.  It is stateless beyond the surface Y assumption used for
// heightmaps.
//
// The surface Y (currently hardcoded to 63, matching FlatGenerator) will
// become per-chunk once region-file loading is implemented.
type Sender struct {
	// SurfaceY is the absolute Y coordinate of the highest solid block, used
	// to compute MOTION_BLOCKING and WORLD_SURFACE heightmaps.
	// Defaults to 63 (one block below the Y=64 spawn in FlatGenerator).
	SurfaceY int
}

// DefaultSender is a Sender pre-configured for the flat world (surface at Y=63).
var DefaultSender = &Sender{SurfaceY: 63}

// SendChunk encodes c and sends it to conn as a Level Chunk With Light packet.
//
// Packet layout (1.21.4):
//
//	Int        Chunk X
//	Int        Chunk Z
//	NBT        Heightmaps (MOTION_BLOCKING + WORLD_SURFACE long arrays)
//	ByteArray  Data (24 encoded sections)
//	VarInt     Block entity count (0 for M4)
//	— Light update fields —
//	BitSet     Sky Light Mask   (all 26 sections → fully lit)
//	BitSet     Block Light Mask (empty)
//	BitSet     Empty Sky Light Mask  (empty — we provide actual data)
//	BitSet     Empty Block Light Mask (all 26 → zero block light)
//	VarInt     Sky Light array count (26)
//	ByteArray  × 26  (each 2048 bytes of 0xFF = max sky light)
//	VarInt     Block Light array count (0)
func (s *Sender) SendChunk(conn *network.ClientConn, c *coreworld.Chunk) error {
	heightmapNBT := EncodeHeightmaps(s.SurfaceY)
	chunkData := EncodeChunk(c)

	// All-sections sky light mask: 26 bits set (sections -1 to 24 inclusive).
	// Stored as a single long in the BitSet wire format: VarInt(1) + Long(mask).
	const allSectionsMask = int64((int64(1) << 26) - 1) // 0x3FFFFFF

	// Build a sky light array: 2048 bytes of 0xFF (nibble pairs, each = 15 = max).
	fullLight := make([]byte, 2048)
	for i := range fullLight {
		fullLight[i] = 0xFF
	}

	b := protocol.NewBuilder(packetIDLevelChunkWithLight).
		Int(c.X).
		Int(c.Z).
		// Heightmaps NBT (self-delimiting, no length prefix)
		Bytes(heightmapNBT).
		// Chunk section data (VarInt-prefixed byte array)
		ByteArray(chunkData).
		// Block entities
		VarInt(0).
		// Sky Light Mask BitSet: 1 long, all 26 section bits set
		VarInt(1).Long(allSectionsMask).
		// Block Light Mask BitSet: empty (no block light data provided)
		VarInt(0).
		// Empty Sky Light Mask BitSet: empty (we ARE providing sky light data)
		VarInt(0).
		// Empty Block Light Mask BitSet: all 26 sections have zero block light
		VarInt(1).Long(allSectionsMask).
		// Sky Light arrays: 26 × 2048 bytes of 0xFF
		VarInt(26)

	for i := 0; i < 26; i++ {
		b.ByteArray(fullLight)
	}

	// Block Light arrays: none
	b.VarInt(0)

	return conn.WritePacket(b.Build())
}

// SendChunksAround sends all chunks in a square of radius r centred on chunk
// (cx, cz).  Chunks are fetched from w (generating them if necessary) and
// sent in spiral order — centre first, then outward — so the player sees their
// immediate surroundings first.
func (s *Sender) SendChunksAround(conn *network.ClientConn, w *coreworld.World, cx, cz, r int32) error {
	// Send centre chunk first, then spiral outward.
	for dx := -r; dx <= r; dx++ {
		for dz := -r; dz <= r; dz++ {
			c := w.Chunk(cx+int32(dx), cz+int32(dz))
			if err := s.SendChunk(conn, c); err != nil {
				return fmt.Errorf("sending chunk (%d,%d): %w", cx+dx, cz+dz, err)
			}
		}
	}
	return nil
}
