package handler

import (
	"fmt"
	"strings"

	coreworld "GoCraft/core/world"
)

type clonedBlock struct {
	dx, dy, dz int
	block      coreworld.Block
}

func cmdClone(ctx CommandContext) error {
	if ctx.Player == nil || ctx.World == nil {
		return fmt.Errorf("world state is unavailable")
	}
	if len(ctx.Args) < 9 || len(ctx.Args) > 11 {
		return fmt.Errorf("usage: /clone <begin> <end> <destination> [replace|masked] [normal|force|move]")
	}
	coords := make([]int, 9)
	for i := range coords {
		origin := []float64{ctx.Player.Position.X, ctx.Player.Position.Y, ctx.Player.Position.Z}[i%3]
		value, err := ParseCommandCoordinate(ctx.Args[i], origin)
		if err != nil {
			return err
		}
		coords[i] = value
	}
	mask, behavior := "replace", "normal"
	if len(ctx.Args) >= 10 {
		mask = strings.ToLower(ctx.Args[9])
	}
	if len(ctx.Args) == 11 {
		behavior = strings.ToLower(ctx.Args[10])
	}
	if mask != "replace" && mask != "masked" {
		return fmt.Errorf("unknown clone mask: %s", mask)
	}
	if behavior != "normal" && behavior != "force" && behavior != "move" {
		return fmt.Errorf("unknown clone behavior: %s", behavior)
	}
	minX, maxX := min(coords[0], coords[3]), max(coords[0], coords[3])
	minY, maxY := min(coords[1], coords[4]), max(coords[1], coords[4])
	minZ, maxZ := min(coords[2], coords[5]), max(coords[2], coords[5])
	volume := int64(maxX-minX+1) * int64(maxY-minY+1) * int64(maxZ-minZ+1)
	if volume > 32768 {
		return fmt.Errorf("too many blocks: %d (maximum 32768)", volume)
	}
	destinationMax := [3]int{coords[6] + maxX - minX, coords[7] + maxY - minY, coords[8] + maxZ - minZ}
	if behavior == "normal" && boxesOverlap(minX, minY, minZ, maxX, maxY, maxZ,
		coords[6], coords[7], coords[8], destinationMax[0], destinationMax[1], destinationMax[2]) {
		return fmt.Errorf("source and destination overlap; use force or move")
	}
	blocks := make([]clonedBlock, 0, volume)
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			for z := minZ; z <= maxZ; z++ {
				block := ctx.World.GetBlock(x, y, z)
				if mask != "masked" || !block.IsAir() {
					blocks = append(blocks, clonedBlock{x - minX, y - minY, z - minZ, block})
				}
				if behavior == "move" {
					ctx.World.SetBlock(x, y, z, coreworld.Air)
				}
			}
		}
	}
	for _, copied := range blocks {
		ctx.World.SetBlock(coords[6]+copied.dx, coords[7]+copied.dy, coords[8]+copied.dz, copied.block)
	}
	return sendCommandMessage(ctx, fmt.Sprintf("Cloned %d blocks", len(blocks)))
}

func boxesOverlap(ax1, ay1, az1, ax2, ay2, az2, bx1, by1, bz1, bx2, by2, bz2 int) bool {
	return ax1 <= bx2 && ax2 >= bx1 && ay1 <= by2 && ay2 >= by1 && az1 <= bz2 && az2 >= bz1
}
