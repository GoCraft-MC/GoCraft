package world

import "strconv"

const NoteBlockPitchCount = 25

// TuneNoteBlock advances a note block by one semitone and wraps after F sharp.
func TuneNoteBlock(block Block) (Block, bool) {
	if block.ResourceLocation() != "minecraft:note_block" {
		return Block{}, false
	}
	note, err := strconv.Atoi(block.Properties["note"])
	if err != nil || note < 0 || note >= NoteBlockPitchCount {
		note = 0
	}
	updated := copyInteractionBlock(block)
	updated.Properties["note"] = strconv.Itoa((note + 1) % NoteBlockPitchCount)
	return updated, true
}
