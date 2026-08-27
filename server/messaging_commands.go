package server

import (
	"fmt"
	"strings"

	"GoCraft/core/player"
	"GoCraft/java/handler"
)

func (s *Server) registerMessagingCommands() {
	s.cmds.Register("me", func(ctx handler.CommandContext) error {
		if ctx.Player == nil || len(ctx.Args) == 0 {
			return fmt.Errorf("usage: /me <action>")
		}
		s.broadcastMessage("* " + ctx.Player.Username + " " + strings.Join(ctx.Args, " "))
		return nil
	})
	s.cmds.RegisterOperator("say", func(ctx handler.CommandContext) error {
		if len(ctx.Args) == 0 {
			return fmt.Errorf("usage: /say <message>")
		}
		s.broadcastMessage("[Server] " + strings.Join(ctx.Args, " "))
		return nil
	})
	for _, name := range []string{"msg", "tell", "w"} {
		s.cmds.Register(name, s.commandPrivateMessage)
	}
}

func (s *Server) commandPrivateMessage(ctx handler.CommandContext) error {
	if ctx.Player == nil || len(ctx.Args) < 2 {
		return fmt.Errorf("usage: /msg <player> <message>")
	}
	target := s.findOnlinePlayer(ctx.Args[0])
	if target == nil {
		return fmt.Errorf("player not found: %s", ctx.Args[0])
	}
	message := strings.Join(ctx.Args[1:], " ")
	if err := s.sendPlayerMessage(target, "["+ctx.Player.Username+" -> you] "+message); err != nil {
		return err
	}
	return commandReply(ctx, "[you -> "+target.Username+"] "+message)
}

func (s *Server) findOnlinePlayer(name string) *player.Player {
	var found *player.Player
	if s.game != nil {
		s.game.OnlinePlayers(func(candidate *player.Player) {
			if found == nil && strings.EqualFold(candidate.Username, name) {
				found = candidate
			}
		})
	}
	return found
}

func (s *Server) sendPlayerMessage(target *player.Player, message string) error {
	if target.Edition == player.ClientEditionJava {
		if current, ok := s.sessions.Get(target.UUID); ok {
			return handler.SendSystemMessage(current.Conn, message)
		}
	} else if s.bedrockListener != nil {
		s.bedrockListener.SendMessage(target.UUID, message)
		return nil
	}
	return fmt.Errorf("player session is unavailable")
}

func (s *Server) broadcastMessage(message string) {
	handler.BroadcastSystemMessage(s.sessions, message)
}
