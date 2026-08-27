package handler

import (
	"fmt"
	"strings"

	coreworld "GoCraft/core/world"
)

func cmdFill(ctx CommandContext) error {
	if ctx.Player == nil || ctx.World == nil {
		return fmt.Errorf("world state is unavailable")
	}
	if len(ctx.Args) < 7 || len(ctx.Args) > 9 {
		return fmt.Errorf("usage: /fill <from> <to> <block> [replace [filter]|keep|hollow|outline]")
	}
	coords := make([]int, 6)
	origins := []float64{ctx.Player.Position.X, ctx.Player.Position.Y, ctx.Player.Position.Z,
		ctx.Player.Position.X, ctx.Player.Position.Y, ctx.Player.Position.Z}
	for i := range coords {
		value, err := ParseCommandCoordinate(ctx.Args[i], origins[i])
		if err != nil {
			return err
		}
		coords[i] = value
	}
	block, err := parseCommandBlock(ctx.Args[6])
	if err != nil {
		return err
	}
	mode := "replace"
	if len(ctx.Args) >= 8 {
		mode = strings.ToLower(ctx.Args[7])
	}
	if mode != "replace" && mode != "keep" && mode != "hollow" && mode != "outline" {
		return fmt.Errorf("unknown fill mode: %s", mode)
	}
	var filter *coreworld.Block
	if len(ctx.Args) == 9 {
		if mode != "replace" {
			return fmt.Errorf("a block filter requires replace mode")
		}
		parsed, err := parseCommandBlock(ctx.Args[8])
		if err != nil {
			return err
		}
		filter = &parsed
	}
	minX, maxX := min(coords[0], coords[3]), max(coords[0], coords[3])
	minY, maxY := min(coords[1], coords[4]), max(coords[1], coords[4])
	minZ, maxZ := min(coords[2], coords[5]), max(coords[2], coords[5])
	volume := int64(maxX-minX+1) * int64(maxY-minY+1) * int64(maxZ-minZ+1)
	if volume > 32768 {
		return fmt.Errorf("too many blocks: %d (maximum 32768)", volume)
	}
	changed := 0
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				current := ctx.World.GetBlock(x, y, z)
				boundary := x == minX || x == maxX || y == minY || y == maxY || z == minZ || z == maxZ
				replacement, ok := fillReplacement(mode, boundary, current, block, filter)
				if ok && !current.Equal(replacement) {
					ctx.World.SetBlock(x, y, z, replacement)
					changed++
				}
			}
		}
	}
	return sendCommandMessage(ctx, fmt.Sprintf("Filled %d blocks", changed))
}

func fillReplacement(mode string, boundary bool, current, block coreworld.Block, filter *coreworld.Block) (coreworld.Block, bool) {
	if mode == "keep" && !current.IsAir() || mode == "outline" && !boundary {
		return coreworld.Block{}, false
	}
	if filter != nil && !current.Equal(*filter) {
		return coreworld.Block{}, false
	}
	if mode == "hollow" && !boundary {
		return coreworld.Air, true
	}
	return block, true
}
