package handler

import (
	"fmt"
	"strconv"

	"GoCraft/core/player"
	javaworld "GoCraft/java/world"
)

func cmdClear(ctx CommandContext) error {
	if ctx.Player == nil {
		return fmt.Errorf("player state is unavailable")
	}
	if len(ctx.Args) > 3 {
		return fmt.Errorf("usage: /clear [player] [item] [max-count]")
	}
	target := ctx.Player
	if len(ctx.Args) >= 1 && ctx.Args[0] != "@s" {
		target = findCanonicalPlayer(ctx, ctx.Args[0])
		if target == nil {
			return fmt.Errorf("player not found: %s", ctx.Args[0])
		}
	}
	item := ""
	if len(ctx.Args) >= 2 {
		item = normalizeResourceLocation(ctx.Args[1])
		if javaworld.ItemID(item) < 0 {
			return fmt.Errorf("unknown item: %s", ctx.Args[1])
		}
	}
	maximum := int(^uint(0) >> 1)
	if len(ctx.Args) == 3 {
		parsed, err := strconv.Atoi(ctx.Args[2])
		if err != nil || parsed < 0 {
			return fmt.Errorf("max-count must be a non-negative integer")
		}
		maximum = parsed
	}
	removed := 0
	for slot := range target.Inventory {
		stack := &target.Inventory[slot]
		if stack.IsEmpty() || item != "" && stack.ItemID != item || removed >= maximum {
			continue
		}
		amount := min(stack.Count, maximum-removed)
		stack.Count -= amount
		removed += amount
		if stack.Count == 0 {
			*stack = player.ItemStack{}
		}
	}
	SendPlayerInventory(target, ctx.Manager)
	return sendCommandMessage(ctx, fmt.Sprintf("Removed %d items from %s", removed, target.Username))
}
