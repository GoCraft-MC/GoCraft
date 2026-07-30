package world

// Generator creates chunks for positions not yet loaded or generated.
// Implementations must be safe for concurrent calls.
type Generator interface {
	Generate(x, z int32) *Chunk
}

// FlatGenerator produces a minimal flat world with a single layer of stone
// at Y=63 (one block below the default spawn at Y=64).
//
// This is the Milestone 4 placeholder; future milestones will add region-file
// loading, configurable presets, and proper layering.
type FlatGenerator struct{}

// Generate returns a chunk with stone at Y=63 and air everywhere else.
func (g *FlatGenerator) Generate(x, z int32) *Chunk {
	c := &Chunk{X: x, Z: z}

	// Section index for Y=63: (63 - WorldMinY) / SectionSize = 127 / 16 = 7
	// Local Y within section 7: 127 % 16 = 15  (absolute Y = 48+15 = 63)
	const groundSectionIdx = 7
	const groundLocalY = 15

	sec := NewSection()
	sec.Biome = "minecraft:plains"
	for bx := 0; bx < SectionSize; bx++ {
		for bz := 0; bz < SectionSize; bz++ {
			sec.Set(bx, groundLocalY, bz, "minecraft:stone")
		}
	}
	c.Sections[groundSectionIdx] = sec

	return c
}
