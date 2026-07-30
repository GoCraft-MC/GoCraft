// Package world contains the edition-agnostic world model for GoCraft.
// Java- and Bedrock-specific packet structures must NOT appear here.
// The Java adapter (java/world) converts canonical types to wire format;
// a future Bedrock adapter will do the same independently.
package world

// BlockState identifies a block by its canonical Minecraft resource location.
// Future milestones will add a Properties map for block state variants
// (e.g. {"facing":"north","waterlogged":"false"}).
type BlockState struct {
	// Name is the canonical Minecraft identifier, e.g. "minecraft:stone".
	Name string
}

// Air is the canonical air block used to represent empty space.
var Air = BlockState{Name: "minecraft:air"}

// IsAir returns true for the air block or the zero value.
func (b BlockState) IsAir() bool {
	return b.Name == "" || b.Name == "minecraft:air"
}
