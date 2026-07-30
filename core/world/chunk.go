package world

const (
	// SectionSize is the number of blocks along each axis of a chunk section.
	SectionSize = 16

	// WorldMinY is the minimum world Y coordinate in 1.18+ (post height expansion).
	WorldMinY = -64
	// WorldMaxY is the maximum world Y coordinate (inclusive) in 1.18+.
	WorldMaxY = 319

	// SectionCount is the number of 16-block-tall sections per chunk column.
	// (WorldMaxY - WorldMinY + 1) / SectionSize = 384 / 16 = 24
	SectionCount = (WorldMaxY - WorldMinY + 1) / SectionSize
)

// Chunk is the canonical, edition-agnostic chunk: a 16×(24×16)×16 column of blocks.
//
// The Java adapter (java/world) converts this to the Level Chunk With Light packet.
// A future Bedrock adapter will convert it to a Sub Chunk packet independently.
// Neither the field types nor the method signatures must reference Java or Bedrock.
type Chunk struct {
	// X and Z are chunk coordinates (block position divided by 16).
	X, Z int32
	// Sections holds one Section per 16-block-tall slice, bottom first.
	// A nil entry represents an all-air section (saves memory for empty columns).
	Sections [SectionCount]*Section
}

// SectionMinY returns the absolute minimum Y coordinate of section index i.
func SectionMinY(i int) int { return WorldMinY + i*SectionSize }

// Section is one 16×16×16 cube of blocks within a chunk.
//
// Internally it uses a palette so that only the block names that actually
// appear are stored; all positions hold a 16-bit palette index.
// Index 0 in the palette is always "minecraft:air".
//
// The Java encoder (java/world/encoder.go) reads BlockPalette() and BlockData()
// to build the PalettedContainer wire format without touching this package's
// internals — keeping the canonical and Java-specific layers separate.
type Section struct {
	blockPalette []string     // canonical block names; [0] = "minecraft:air"
	blockData    [4096]uint16 // palette index per block: y*256 + z*16 + x
	NonAir       int16        // count of non-air blocks (required for Java wire format)

	// Biome is the canonical biome name for all 64 biome cells in this section.
	// A future milestone will store per-cell biome data.
	Biome string
}

// NewSection returns an all-air section with a single-entry palette.
func NewSection() *Section {
	return &Section{
		blockPalette: []string{"minecraft:air"},
		Biome:        "minecraft:plains",
	}
}

// At returns the canonical block name at section-local coordinates (0–15 each).
// Returns "minecraft:air" for nil sections.
func (s *Section) At(x, y, z int) string {
	if s == nil || len(s.blockPalette) == 0 {
		return "minecraft:air"
	}
	return s.blockPalette[s.blockData[y*256+z*16+x]]
}

// Set places a block at section-local coordinates.
// It grows the palette on first use of a block name.
func (s *Section) Set(x, y, z int, name string) {
	idx := y*256 + z*16 + x

	// Search existing palette entries.
	for i, p := range s.blockPalette {
		if p == name {
			old := s.blockData[idx]
			s.blockData[idx] = uint16(i)
			s.updateNonAir(old, uint16(i))
			return
		}
	}

	// New block name — append to palette.
	s.blockPalette = append(s.blockPalette, name)
	newIdx := uint16(len(s.blockPalette) - 1)
	old := s.blockData[idx]
	s.blockData[idx] = newIdx
	s.updateNonAir(old, newIdx)
}

// updateNonAir adjusts the NonAir counter after a palette index change.
// Palette index 0 is always air.
func (s *Section) updateNonAir(old, new uint16) {
	wasAir := old == 0
	isAir := new == 0
	if wasAir && !isAir {
		s.NonAir++
	} else if !wasAir && isAir {
		s.NonAir--
	}
}

// BlockPalette returns the canonical block name palette.
// The Java encoder uses this to look up Java global state IDs.
func (s *Section) BlockPalette() []string { return s.blockPalette }

// BlockData returns the raw palette-index array (position: y*256+z*16+x).
// The Java encoder uses this to pack the PalettedContainer bit array.
func (s *Section) BlockData() [4096]uint16 { return s.blockData }
