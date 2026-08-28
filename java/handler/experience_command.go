package handler

import (
	"fmt"
	"strconv"
	"strings"

	"GoCraft/core/player"
)

func cmdExperience(ctx CommandContext) error {
	if len(ctx.Args) < 3 {
		return fmt.Errorf("usage: /experience <add|set|query> <player> <amount|points|levels> [points|levels]")
	}
	target := findCanonicalPlayer(ctx, ctx.Args[1])
	if target == nil {
		return fmt.Errorf("player not found: %s", ctx.Args[1])
	}
	action := strings.ToLower(ctx.Args[0])
	if action == "query" {
		if len(ctx.Args) != 3 {
			return fmt.Errorf("usage: /experience query <player> <points|levels>")
		}
		level, total, _ := target.ExperienceSnapshot()
		value := level
		if strings.EqualFold(ctx.Args[2], "points") {
			value = total - player.ExperienceForLevel(level)
		} else if !strings.EqualFold(ctx.Args[2], "levels") {
			return fmt.Errorf("experience unit must be points or levels")
		}
		return sendCommandMessage(ctx, fmt.Sprintf("%s has %d %s", target.Username, value, ctx.Args[2]))
	}
	if len(ctx.Args) < 3 || len(ctx.Args) > 4 {
		return fmt.Errorf("usage: /experience <add|set> <player> <amount> [points|levels]")
	}
	amount, err := strconv.ParseInt(ctx.Args[2], 10, 32)
	if err != nil {
		return fmt.Errorf("amount must be an integer")
	}
	unit := "points"
	if len(ctx.Args) == 4 {
		unit = strings.ToLower(ctx.Args[3])
	}
	if unit != "points" && unit != "levels" {
		return fmt.Errorf("experience unit must be points or levels")
	}
	switch action {
	case "add":
		if unit == "points" {
			target.AddExperience(int32(amount))
		} else {
			level, _, _ := target.ExperienceSnapshot()
			target.SetExperienceLevel(level + int32(amount))
		}
	case "set":
		if amount < 0 {
			return fmt.Errorf("amount must be non-negative")
		}
		if unit == "levels" {
			target.SetExperienceLevel(int32(amount))
		} else {
			level, _, _ := target.ExperienceSnapshot()
			if amount >= int64(player.ExperienceToNextLevel(level)) {
				return fmt.Errorf("points must be less than %d at level %d", player.ExperienceToNextLevel(level), level)
			}
			target.SetTotalExperience(player.ExperienceForLevel(level) + int32(amount))
		}
	default:
		return fmt.Errorf("usage: /experience <add|set|query> ...")
	}
	SyncPlayerExperience(target, ctx.Manager)
	level, total, _ := target.ExperienceSnapshot()
	return sendCommandMessage(ctx, fmt.Sprintf("%s now has level %d (%d total points)", target.Username, level, total))
}
