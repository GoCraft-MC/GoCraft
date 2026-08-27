package server

import (
	"fmt"
	"strings"

	"GoCraft/core/player"
	"GoCraft/java/handler"
)

func (s *Server) registerWorldSettingsCommands() {
	s.cmds.RegisterOperator("defaultgamemode", s.commandDefaultGameMode)
	s.cmds.RegisterOperator("difficulty", s.commandDifficulty)
}

func (s *Server) commandDefaultGameMode(ctx handler.CommandContext) error {
	if len(ctx.Args) != 1 {
		return fmt.Errorf("usage: /defaultgamemode <survival|creative|adventure|spectator>")
	}
	var mode player.GameMode
	switch strings.ToLower(ctx.Args[0]) {
	case "survival":
		mode = player.GameModeSurvival
	case "creative":
		mode = player.GameModeCreative
	case "adventure":
		mode = player.GameModeAdventure
	case "spectator":
		mode = player.GameModeSpectator
	default:
		return fmt.Errorf("unknown game mode: %s", ctx.Args[0])
	}
	s.defaultGameMode.Store(uint32(mode))
	if s.bedrockListener != nil {
		s.bedrockListener.SetDefaultGameMode(mode)
	}
	return commandReply(ctx, "Default game mode set to "+strings.ToLower(ctx.Args[0]))
}

func (s *Server) commandDifficulty(ctx handler.CommandContext) error {
	names := []string{"peaceful", "easy", "normal", "hard"}
	if len(ctx.Args) == 0 {
		return commandReply(ctx, "Difficulty is "+names[s.currentDifficulty()])
	}
	if len(ctx.Args) != 1 {
		return fmt.Errorf("usage: /difficulty [peaceful|easy|normal|hard]")
	}
	requested := strings.ToLower(ctx.Args[0])
	level := int32(-1)
	for index, name := range names {
		if requested == name {
			level = int32(index)
		}
	}
	if level < 0 {
		return fmt.Errorf("unknown difficulty: %s", requested)
	}
	s.difficulty.Store(level + 1)
	if s.bedrockListener != nil {
		s.bedrockListener.SetDifficulty(level)
	}
	return commandReply(ctx, "Difficulty set to "+requested)
}
