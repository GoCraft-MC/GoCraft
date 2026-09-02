package server

import (
	"log/slog"

	"GoCraft/core/dispatch"
	"GoCraft/core/player"
)

// pluginCommandSender is whoever typed a plugin command, as core/dispatch sees
// them.
//
// It carries the player rather than a connection on purpose: that is the
// blocker §18 names, and it is what makes one command work on Java, Bedrock and
// the console at once. Each of the three answers differ only inside the two
// methods below, and neither of them knows which edition it is serving —
// sendPlayerMessage already does, and the permission manager never cared.
type pluginCommandSender struct {
	server *Server
	player *player.Player
}

// consoleSenderName is what a handler naming its sender prints when nobody is
// behind the command.
const consoleSenderName = "Console"

func (s *Server) commandSenderFor(p *player.Player) dispatch.Sender {
	return pluginCommandSender{server: s, player: p}
}

// consoleCommandSender is the server itself: no player, every permission, and
// output to the log.
func (s *Server) consoleCommandSender() dispatch.Sender {
	return pluginCommandSender{server: s}
}

func (s pluginCommandSender) Name() string {
	if s.player == nil {
		return consoleSenderName
	}
	return s.player.Username
}

func (s pluginCommandSender) UUID() [16]byte {
	if s.player == nil {
		return [16]byte{}
	}
	return s.player.UUID
}

// Has answers from the same permission manager the event bus injects its
// resolved nodes from, so a plugin cannot be told two different things about
// one player depending on whether it asked during an event or a command.
//
// The console holds everything: it is the operator's own terminal, and a server
// that could refuse it a command would be refusing its own administrator.
func (s pluginCommandSender) Has(node string) bool {
	if s.player == nil {
		return true
	}
	if s.server == nil || s.server.permissions == nil {
		return s.player.Operator
	}
	return s.server.permissions.Allowed(s.player.Username, node, s.player.Operator, false)
}

func (s pluginCommandSender) SendMessage(message string) error {
	if s.player == nil {
		slog.Info("command", "sender", consoleSenderName, "message", message)
		return nil
	}
	return s.server.sendPlayerMessage(s.player, message)
}

func (s pluginCommandSender) Player() (*player.Player, bool) {
	return s.player, s.player != nil
}
