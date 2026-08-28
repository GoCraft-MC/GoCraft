package handler

import (
	"fmt"
	"strings"
)

func cmdBanList(ctx CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("usage: /banlist [ips|players]")
	}
	typeName := "players"
	if len(ctx.Args) == 1 {
		typeName = strings.ToLower(ctx.Args[0])
	}
	var values []string
	switch typeName {
	case "players":
		values = BannedPlayers()
	case "ips":
		values = BannedIPs()
	default:
		return fmt.Errorf("usage: /banlist [ips|players]")
	}
	if len(values) == 0 {
		return sendCommandMessage(ctx, "There are no banned "+typeName)
	}
	return sendCommandMessage(ctx, "Banned "+typeName+": "+strings.Join(values, ", "))
}

func cmdDeop(ctx CommandContext) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("usage: /deop <player>")
	}
	removed, err := RemoveOperator(ctx.Args[0])
	if err != nil {
		return fmt.Errorf("saving ops.json: %w", err)
	}
	if !removed {
		return fmt.Errorf("%s is not an operator", ctx.Args[0])
	}
	if target := findCommandPlayer(ctx, ctx.Args[0]); target != nil {
		target.Operator = false
		if ctx.SyncAbilities != nil {
			ctx.SyncAbilities(target)
		}
	}
	return sendCommandMessage(ctx, "Removed operator status from "+ctx.Args[0])
}
