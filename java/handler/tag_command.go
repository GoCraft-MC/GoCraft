package handler

import (
	"fmt"
	"strings"
)

func cmdTag(ctx CommandContext) error {
	if len(ctx.Args) < 2 || len(ctx.Args) > 3 {
		return fmt.Errorf("usage: /tag <player> <add|remove|list> [tag]")
	}
	target := findCanonicalPlayer(ctx, ctx.Args[0])
	if target == nil {
		return fmt.Errorf("player not found: %s", ctx.Args[0])
	}
	action := strings.ToLower(ctx.Args[1])
	if action == "list" {
		if len(ctx.Args) != 2 {
			return fmt.Errorf("usage: /tag <player> list")
		}
		tags := target.Tags()
		if len(tags) == 0 {
			return sendCommandMessage(ctx, target.Username+" has no tags")
		}
		return sendCommandMessage(ctx, target.Username+" has tags: "+strings.Join(tags, ", "))
	}
	if len(ctx.Args) != 3 {
		return fmt.Errorf("usage: /tag <player> <add|remove> <tag>")
	}
	switch action {
	case "add":
		if !target.AddTag(ctx.Args[2]) {
			return fmt.Errorf("tag is invalid or already present")
		}
		return sendCommandMessage(ctx, "Added tag "+ctx.Args[2]+" to "+target.Username)
	case "remove":
		if !target.RemoveTag(ctx.Args[2]) {
			return fmt.Errorf("tag is not present")
		}
		return sendCommandMessage(ctx, "Removed tag "+ctx.Args[2]+" from "+target.Username)
	default:
		return fmt.Errorf("usage: /tag <player> <add|remove|list> [tag]")
	}
}
