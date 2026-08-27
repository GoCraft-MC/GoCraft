package handler

import (
	"fmt"
	"strings"
)

func cmdSetBlock(ctx CommandContext) error {
	if ctx.Player == nil || ctx.World == nil {
		return fmt.Errorf("world state is unavailable")
	}
	if len(ctx.Args) < 4 || len(ctx.Args) > 5 {
		return fmt.Errorf("usage: /setblock <x> <y> <z> <block> [replace|keep|destroy]")
	}
	x, err := parseCommandCoordinate(ctx.Args[0], ctx.Player.Position.X)
	if err != nil {
		return err
	}
	y, err := parseCommandCoordinate(ctx.Args[1], ctx.Player.Position.Y)
	if err != nil {
		return err
	}
	z, err := parseCommandCoordinate(ctx.Args[2], ctx.Player.Position.Z)
	if err != nil {
		return err
	}
	block, err := parseCommandBlock(ctx.Args[3])
	if err != nil {
		return err
	}
	mode := "replace"
	if len(ctx.Args) == 5 {
		mode = strings.ToLower(ctx.Args[4])
	}
	if mode != "replace" && mode != "keep" && mode != "destroy" {
		return fmt.Errorf("unknown setblock mode: %s", mode)
	}
	if mode == "keep" && !ctx.World.GetBlock(x, y, z).IsAir() {
		return fmt.Errorf("could not place block")
	}
	ctx.World.SetBlock(x, y, z, block)
	return sendCommandMessage(ctx, "Changed the block")
}
