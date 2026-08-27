package handler

import (
	"fmt"
	"strings"

	"GoCraft/core/player"
)

func registerModerationCommands(d *Dispatcher) {
	d.RegisterOperator("ban", cmdBan)
	d.RegisterOperator("ban-ip", cmdBanIP)
	d.RegisterOperator("pardon", cmdPardon)
	d.RegisterOperator("pardon-ip", cmdPardonIP)
}

func commandSource(ctx CommandContext) string {
	if ctx.Player == nil {
		return "Server"
	}
	return ctx.Player.Username
}

func cmdBan(ctx CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: /ban <player> [reason]")
	}
	name, reason := ctx.Args[0], defaultBanReason(strings.Join(ctx.Args[1:], " "))
	if err := BanPlayer(name, commandSource(ctx), reason); err != nil {
		return fmt.Errorf("saving player ban: %w", err)
	}
	if target := findCommandPlayer(ctx, name); target != nil && ctx.DisconnectPlayer != nil {
		_ = ctx.DisconnectPlayer(target, "You are banned: "+reason)
	}
	return sendCommandMessage(ctx, fmt.Sprintf("Banned %s: %s", name, reason))
}

func cmdPardon(ctx CommandContext) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("usage: /pardon <player>")
	}
	removed, err := PardonPlayer(ctx.Args[0])
	if err != nil {
		return fmt.Errorf("saving player bans: %w", err)
	}
	if !removed {
		return fmt.Errorf("%s is not banned", ctx.Args[0])
	}
	return sendCommandMessage(ctx, "Pardoned "+ctx.Args[0])
}

func findCommandPlayer(ctx CommandContext, name string) *player.Player {
	if ctx.FindPlayer == nil {
		return nil
	}
	return ctx.FindPlayer(name)
}
