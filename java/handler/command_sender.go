package handler

import (
	"strings"

	"GoCraft/core/command"
	"GoCraft/core/player"
)

// dispatcherSender is a player as the command registry judges them.
//
// It exists because a built-in and a plugin command are permitted by different
// rules. A built-in carries a default: /help is allowed unless an admin takes
// it away, /stop is refused unless one grants it, and that default is per
// command rather than derivable from the node. A plugin node has no default —
// it was never granted, so it is denied. One Has method cannot guess which it
// is being asked about, so this one looks the node up.
type dispatcherSender struct {
	dispatcher *Dispatcher
	player     *player.Player
}

func (s dispatcherSender) Name() string {
	if s.player == nil {
		return "Console"
	}
	return s.player.Username
}

func (s dispatcherSender) UUID() [16]byte {
	if s.player == nil {
		return [16]byte{}
	}
	return s.player.UUID
}

// Has answers a built-in exactly as CanUse does, and hands anything else to the
// plugin permission bridge.
func (s dispatcherSender) Has(node string) bool {
	if name, ok := builtinCommandName(node); ok {
		return s.dispatcher.CanUse(s.player, name)
	}
	s.dispatcher.mu.RLock()
	check := s.dispatcher.permission
	s.dispatcher.mu.RUnlock()
	if check == nil {
		return s.player != nil && s.player.Operator
	}
	return check(s.player, node, false)
}

func (s dispatcherSender) SendMessage(message string) error {
	s.dispatcher.mu.RLock()
	send := s.dispatcher.messenger
	s.dispatcher.mu.RUnlock()
	if send == nil || s.player == nil {
		return nil
	}
	return send(s.player, message)
}

func (s dispatcherSender) Player() (*player.Player, bool) {
	return s.player, s.player != nil
}

// Sender wraps a player so the command registry can judge what they may see.
func (d *Dispatcher) Sender(p *player.Player) command.Sender {
	return dispatcherSender{dispatcher: d, player: p}
}

// CommandTree is every command this player may use, built-ins and plugins in
// one tree.
//
// One snapshot, so both editions render the same thing and neither has to know
// how the other decides what is visible. Empty until a registry is installed,
// which is what keeps a dispatcher used on its own in a test from needing one.
func (d *Dispatcher) CommandTree(p *player.Player) command.Root {
	d.mu.RLock()
	registry := d.registry
	d.mu.RUnlock()
	if registry == nil {
		return command.Root{}
	}
	return registry.Snapshot(d.Sender(p)).Root
}

// TreeVersion moves whenever the commands a client should be told about change.
func (d *Dispatcher) TreeVersion() uint64 {
	d.mu.RLock()
	registry := d.registry
	d.mu.RUnlock()
	if registry == nil {
		return 0
	}
	return registry.Version()
}

// builtinCommandName reads a command name back out of the node guarding it.
func builtinCommandName(node string) (string, bool) {
	name, found := strings.CutPrefix(node, builtinPermissionPrefix)
	return name, found && name != ""
}
