package world

import (
	"fmt"

	coreworld "GoCraft/core/world"
	"GoCraft/internal/protocoldata"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

// packetIDLevelChunkWithLight is the server-to-client Play packet ID for
// "Level Chunk With Light", resolved from the embedded protocol data at init.
var packetIDLevelChunkWithLight = protocoldata.MustCB("play", "minecraft:level_chunk_with_light")

// Sender converts canonical chunks into Java chunk packets. Heightmaps are
// derived from each chunk's actual block columns.
type Sender struct {
	// SurfaceY is retained for source compatibility with older callers.
	// It is deprecated and no longer used during chunk encoding.
	SurfaceY int
}

// DefaultSender encodes canonical chunks for Java Edition.
var DefaultSender = &Sender{SurfaceY: 63}

// SendChunk encodes c and sends it to conn as a Level Chunk With Light packet.
//
// Packet layout (1.21.4):
//
//	Int        Chunk X
//	Int        Chunk Z
//	NBT        Heightmaps (MOTION_BLOCKING + WORLD_SURFACE long arrays)
//	ByteArray  Data (24 encoded sections)
//	VarInt     Block entity count, followed by encoded block entities
//	-- Light update fields --
//	BitSet     Sky Light Mask (only sections containing sky light)
//	BitSet     Block Light Mask (empty)
//	BitSet     Empty Sky Light Mask (sections with zero sky light)
//	BitSet     Empty Block Light Mask (all 26 sections)
//	VarInt     Sky Light array count
//	ByteArray  One 2048-byte nibble array per non-empty sky section
//	VarInt     Block Light array count (0)
func (s *Sender) SendChunk(conn *network.ClientConn, c *coreworld.Chunk) error {
	heightmapNBT := EncodeChunkHeightmaps(c)
	chunkData := EncodeChunk(c)
	skyMask, emptySkyMask, skyArrays := buildSkyLight(c)

	type encodedBlockEntity struct {
		entity coreworld.BlockEntity
		typeID int32
	}
	blockEntities := make([]encodedBlockEntity, 0, len(c.BlockEntities))
	for _, entity := range c.BlockEntities {
		if typeID, ok := BlockEntityTypeID(entity.Type); ok && len(entity.Data) > 0 {
			blockEntities = append(blockEntities, encodedBlockEntity{entity: entity, typeID: typeID})
		}
	}

	const allSectionsMask = int64((int64(1) << 26) - 1)
	b := protocol.NewBuilder(packetIDLevelChunkWithLight).
		Int(c.X).
		Int(c.Z).
		Bytes(heightmapNBT).
		ByteArray(chunkData).
		VarInt(int32(len(blockEntities)))
	for _, encoded := range blockEntities {
		entity := encoded.entity
		packedXZ := byte((entity.X&15)<<4 | (entity.Z & 15))
		b.Byte(packedXZ).Short(int16(entity.Y)).VarInt(encoded.typeID).Bytes(entity.Data)
	}
	b.VarInt(1).Long(skyMask).
		VarInt(0). // block light mask
		VarInt(1).Long(emptySkyMask).
		VarInt(1).Long(allSectionsMask). // empty block light mask
		VarInt(int32(len(skyArrays)))
	for _, light := range skyArrays {
		b.ByteArray(light)
	}
	b.VarInt(0) // no block-light arrays
	return conn.WritePacket(b.Build())
}

// buildSkyLight returns protocol masks and 2048-byte nibble arrays for the 24
// world sections plus the two boundary light sections. Terrain and caves below
// the highest opaque block receive zero sky light; open sky receives level 15.
func buildSkyLight(c *coreworld.Chunk) (skyMask, emptyMask int64, arrays [][]byte) {
	var opaqueTop [coreworld.SectionSize * coreworld.SectionSize]int
	for z := 0; z < coreworld.SectionSize; z++ {
		for x := 0; x < coreworld.SectionSize; x++ {
			top := coreworld.WorldMinY - 1
			for sectionIndex := coreworld.SectionCount - 1; sectionIndex >= 0 && top < coreworld.WorldMinY; sectionIndex-- {
				section := c.Sections[sectionIndex]
				if section == nil || section.NonAir == 0 {
					continue
				}
				for y := coreworld.SectionSize - 1; y >= 0; y-- {
					material := section.At(x, y, z)
					if !isSkyTransparent(material.ResourceLocation()) {
						top = coreworld.SectionMinY(sectionIndex) + y
						break
					}
				}
			}
			opaqueTop[z*coreworld.SectionSize+x] = top
		}
	}

	// Bottom boundary section has no sky light.
	emptyMask |= 1
	for sectionIndex := 0; sectionIndex < coreworld.SectionCount; sectionIndex++ {
		light := make([]byte, 2048)
		nonZero := false
		for y := 0; y < 16; y++ {
			worldY := coreworld.SectionMinY(sectionIndex) + y
			for z := 0; z < 16; z++ {
				for x := 0; x < 16; x++ {
					if worldY <= opaqueTop[z*16+x] {
						continue
					}
					index := y*256 + z*16 + x
					byteIndex := index >> 1
					if index&1 == 0 {
						light[byteIndex] |= 0x0f
					} else {
						light[byteIndex] |= 0xf0
					}
					nonZero = true
				}
			}
		}
		bit := int64(1) << (sectionIndex + 1)
		if nonZero {
			skyMask |= bit
			arrays = append(arrays, light)
		} else {
			emptyMask |= bit
		}
	}

	// Top boundary section is full daylight.
	full := make([]byte, 2048)
	for i := range full {
		full[i] = 0xff
	}
	skyMask |= int64(1) << 25
	arrays = append(arrays, full)
	return skyMask, emptyMask, arrays
}

func isSkyTransparent(resourceLocation string) bool {
	switch resourceLocation {
	case "", "minecraft:air", "minecraft:cave_air", "minecraft:void_air",
		"minecraft:water", "minecraft:glass", "minecraft:ice",
		// Leaves
		"minecraft:oak_leaves", "minecraft:birch_leaves", "minecraft:spruce_leaves",
		"minecraft:acacia_leaves", "minecraft:jungle_leaves",
		"minecraft:dark_oak_leaves", "minecraft:cherry_leaves",
		"minecraft:azalea_leaves", "minecraft:flowering_azalea_leaves",
		"minecraft:mangrove_leaves",
		// Short plants / ground cover (let sky light pass through)
		"minecraft:short_grass", "minecraft:grass", "minecraft:fern",
		"minecraft:dead_bush", "minecraft:seagrass",
		// Tall plants (both halves)
		"minecraft:tall_grass", "minecraft:large_fern",
		"minecraft:tall_seagrass",
		// Flowers
		"minecraft:dandelion", "minecraft:poppy", "minecraft:allium",
		"minecraft:azure_bluet", "minecraft:red_tulip", "minecraft:orange_tulip",
		"minecraft:white_tulip", "minecraft:pink_tulip", "minecraft:oxeye_daisy",
		"minecraft:cornflower", "minecraft:lily_of_the_valley", "minecraft:blue_orchid",
		"minecraft:wither_rose", "minecraft:torchflower", "minecraft:pitcher_plant",
		// Double-tall flowers
		"minecraft:sunflower", "minecraft:lilac", "minecraft:rose_bush", "minecraft:peony",
		// Other transparent / non-full-block
		"minecraft:sugar_cane", "minecraft:bamboo", "minecraft:lily_pad",
		"minecraft:brown_mushroom", "minecraft:red_mushroom",
		"minecraft:vine", "minecraft:moss_carpet",
		"minecraft:torch", "minecraft:wall_torch",
		"minecraft:ladder", "minecraft:rail",
		"minecraft:powered_rail", "minecraft:detector_rail", "minecraft:activator_rail":
		return true
	default:
		return false
	}
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
