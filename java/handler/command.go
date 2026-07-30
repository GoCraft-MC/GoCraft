package handler

// Command dispatcher for Milestone 12.
//
// Dispatcher maps slash-command names to CommandFunc handlers and executes
// them with a CommandContext that bundles every resource a handler might need.
// It is created once at server start, has commands registered via Register,
// and is passed through HandlePlay → playLoop → handleChatPacket so any
// C→S packet that looks like a command reaches it.

import (
	"fmt"
	"strings"
	"sync"

	coreworld "GoCraft/core/world"
	"GoCraft/core/player"
	"GoCraft/java/network"
	"GoCraft/java/session"
)

// CommandContext carries every resource a command handler might need.
// The Conn field is the issuing player's connection; use it for per-player
// feedback.  Manager gives access to all online sessions for commands that
// affect other players (e.g. /kick, /tp <player>).
type CommandContext struct {
	Player  *player.Player
	Conn    *network.ClientConn
	Args    []string // tokens after the command name, split on whitespace
	World   *coreworld.World
	Manager *session.Manager
}

// CommandFunc is the handler signature for a built-in server command.
// A non-nil return value is formatted and sent to the issuing player as a
// system message; it is NOT logged as a server error.
type CommandFunc func(ctx CommandContext) error

// Dispatcher maps command names (lower-case) to their implementations.
// All methods are safe for concurrent use.
type Dispatcher struct {
	mu   sync.RWMutex
	cmds map[string]CommandFunc
}

// NewDispatcher returns an empty, ready-to-use Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{cmds: make(map[string]CommandFunc)}
}

// Register adds fn under the given name.  name is lowercased before storage
// and matched case-insensitively at dispatch time.
func (d *Dispatcher) Register(name string, fn CommandFunc) {
	d.mu.Lock()
	d.cmds[strings.ToLower(name)] = fn
	d.mu.Unlock()
}

// Dispatch parses input (with or without a leading '/'), resolves the command
// name, fills ctx.Args with the remaining tokens, and calls the registered
// handler.
//
// Unknown commands and handler errors are reported to ctx.Conn as a system
// message.  Dispatch itself never returns an error to the caller because
// command failures are player-facing, not server-fatal.
func (d *Dispatcher) Dispatch(input string, ctx CommandContext) {
	input = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/"))
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	name := strings.ToLower(parts[0])
	ctx.Args = parts[1:]

	d.mu.RLock()
	fn, ok := d.cmds[name]
	d.mu.RUnlock()

	if !ok {
		_ = sendSystemMessage(ctx.Conn, fmt.Sprintf("Unknown command: /%s", name))
		return
	}
	if err := fn(ctx); err != nil {
		_ = sendSystemMessage(ctx.Conn, "Error: "+err.Error())
	}
}
