package world

// bellPlacementState mirrors Java 1.21.4 BellBlock#getStateForPlacement.
// Vertical clicks produce floor/ceiling bells. Horizontal clicks prefer a wall
// attachment (double-wall when both opposite faces are sturdy) and fall back to
// floor/ceiling if the clicked wall cannot support the bell.
func bellPlacementState(w *World, block Block, x, y, z int, face int32, rotation int) (Block, bool) {
	playerFacing := attachmentHorizontalFacing(rotation)
	props := map[string]string{"powered": "false"}

	if face == 0 || face == 1 {
		attachment := "floor"
		if face == 0 {
			attachment = "ceiling"
		}
		if !bellAttachmentSurvives(w, x, y, z, playerFacing, attachment) {
			return Block{}, false
		}
		props["attachment"] = attachment
		props["facing"] = playerFacing
		block.Properties = props
		return block, true
	}

	if face >= 2 && face <= 5 {
		facing := oppositeHorizontalFacing(attachmentFacingForFace(face))
		attachment := "single_wall"
		if bellHasOppositeWallSupports(w, x, y, z, facing) {
			attachment = "double_wall"
		}
		if bellAttachmentSurvives(w, x, y, z, facing, attachment) {
			props["attachment"] = attachment
			props["facing"] = facing
			block.Properties = props
			return block, true
		}

		// Vanilla falls back from a failed horizontal wall placement to a floor
		// attachment when possible, otherwise a ceiling attachment.
		for _, fallback := range []string{"floor", "ceiling"} {
			if bellAttachmentSurvives(w, x, y, z, facing, fallback) {
				props["attachment"] = fallback
				props["facing"] = facing
				block.Properties = props
				return block, true
			}
		}
	}
	return Block{}, false
}

// bellAttachmentNeighborState mirrors BellBlock#updateShape for support
// changes. A double-wall bell degrades to a single-wall bell when one side is
// removed; a single/floor/ceiling bell breaks when its sole support disappears.
func bellAttachmentNeighborState(w *World, x, y, z int, block Block) (Block, bool) {
	attachment := block.Properties["attachment"]
	facing := block.Properties["facing"]
	switch attachment {
	case "floor", "ceiling":
		if !bellAttachmentSurvives(w, x, y, z, facing, attachment) {
			return Air, false
		}
		return block, true
	case "single_wall":
		if !bellWallSupportAtFacing(w, x, y, z, facing) {
			return Air, false
		}
		if bellWallSupportAtFacing(w, x, y, z, oppositeHorizontalFacing(facing)) {
			updated := copyAttachmentBlock(block)
			updated.Properties["attachment"] = "double_wall"
			return updated, true
		}
		return block, true
	case "double_wall":
		forward := bellWallSupportAtFacing(w, x, y, z, facing)
		oppositeFacing := oppositeHorizontalFacing(facing)
		back := bellWallSupportAtFacing(w, x, y, z, oppositeFacing)
		switch {
		case forward && back:
			return block, true
		case forward:
			updated := copyAttachmentBlock(block)
			updated.Properties["attachment"] = "single_wall"
			return updated, true
		case back:
			updated := copyAttachmentBlock(block)
			updated.Properties["attachment"] = "single_wall"
			updated.Properties["facing"] = oppositeFacing
			return updated, true
		default:
			return Air, false
		}
	default:
		return Air, false
	}
}

func bellAttachmentSurvives(w *World, x, y, z int, facing, attachment string) bool {
	switch attachment {
	case "floor":
		return attachmentPlacementSupportAt(w, x, y-1, z)
	case "ceiling":
		return attachmentPlacementSupportAt(w, x, y+1, z)
	case "single_wall":
		return bellWallSupportAtFacing(w, x, y, z, facing)
	case "double_wall":
		return bellHasOppositeWallSupports(w, x, y, z, facing)
	default:
		return false
	}
}

func bellHasOppositeWallSupports(w *World, x, y, z int, facing string) bool {
	return bellWallSupportAtFacing(w, x, y, z, facing) &&
		bellWallSupportAtFacing(w, x, y, z, oppositeHorizontalFacing(facing))
}

func bellWallSupportAtFacing(w *World, x, y, z int, facing string) bool {
	dx, dz := horizontalFacingOffset(facing)
	return attachmentPlacementSupportAt(w, x+dx, y, z+dz)
}

func attachmentHorizontalFacing(rotation int) string {
	switch rotation & 15 {
	case 2, 3, 4, 5:
		return "east"
	case 6, 7, 8, 9:
		return "south"
	case 10, 11, 12, 13:
		return "west"
	default:
		return "north"
	}
}

func oppositeHorizontalFacing(facing string) string {
	switch facing {
	case "north":
		return "south"
	case "south":
		return "north"
	case "west":
		return "east"
	case "east":
		return "west"
	default:
		return "north"
	}
}

func copyAttachmentBlock(block Block) Block {
	copy := block
	copy.Properties = make(map[string]string, len(block.Properties))
	for key, value := range block.Properties {
		copy.Properties[key] = value
	}
	return copy
}
