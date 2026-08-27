package handler

import (
	"fmt"
	"strconv"
	"strings"

	"GoCraft/java/session"
)

func cmdDamage(ctx CommandContext) error {
	if len(ctx.Args) < 2 || len(ctx.Args) > 3 {
		return fmt.Errorf("usage: /damage <player> <amount> [damage-type]")
	}
	target := findCanonicalPlayer(ctx, ctx.Args[0])
	if target == nil {
		return fmt.Errorf("player not found: %s", ctx.Args[0])
	}
	amount, err := strconv.ParseFloat(ctx.Args[1], 32)
	if err != nil || amount < 0 || amount > 1_000_000 {
		return fmt.Errorf("amount must be between 0 and 1000000")
	}
	cause := "was killed by magic"
	if len(ctx.Args) == 3 {
		cause = "was killed by " + strings.TrimPrefix(ctx.Args[2], "minecraft:")
	}
	targetSession := &session.Session{Player: target}
	if ctx.Manager != nil {
		if current, ok := ctx.Manager.Get(target.UUID); ok {
			targetSession = current
		}
	}
	DamagePlayer(targetSession, float32(amount), cause, ctx.Manager)
	return sendCommandMessage(ctx, fmt.Sprintf("Applied %.1f damage to %s", amount, target.Username))
}
