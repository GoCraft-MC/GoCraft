package handler

import (
	"fmt"
	"strings"
)

func cmdBanIP(ctx CommandContext) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("usage: /ban-ip <address|player> [reason]")
	}
	address := ctx.Args[0]
	target := findCommandPlayer(ctx, address)
	if target != nil {
		address = target.RemoteAddress
	}
	reason := defaultBanReason(strings.Join(ctx.Args[1:], " "))
	if err := BanIP(address, commandSource(ctx), reason); err != nil {
		return err
	}
	if target != nil && ctx.DisconnectPlayer != nil {
		_ = ctx.DisconnectPlayer(target, "Your IP address is banned: "+reason)
	}
	return sendCommandMessage(ctx, fmt.Sprintf("Banned IP %s: %s", address, reason))
}

func cmdPardonIP(ctx CommandContext) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("usage: /pardon-ip <address>")
	}
	address, err := normalizeIPAddress(ctx.Args[0])
	if err != nil {
		return err
	}
	removed, err := PardonIP(address)
	if err != nil {
		return fmt.Errorf("saving IP bans: %w", err)
	}
	if !removed {
		return fmt.Errorf("%s is not banned", address)
	}
	return sendCommandMessage(ctx, "Pardoned IP "+address)
}
