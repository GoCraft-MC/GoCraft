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

	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
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

	// TeleportTo moves the player to (x, y, z), sends Synchronize Player
	// Position, updates the center-chunk anchor, and streams the destination
	// chunks — all before returning.  Commands that reposition the player
	// (e.g. /tp) must call this instead of mutating Player.Position directly
	// so the client's chunk view is kept in sync.
	TeleportTo func(x, y, z float64) error

	// ChangeWorld moves the issuing player to a vanilla dimension and performs
	// the edition-specific network transition.
	ChangeWorld func(dimension int32) error

	// TeleportPlayer moves an arbitrary online player through the network
	// adapter that owns that player. Cross-edition commands such as /tphere
	// must use this callback instead of mutating the target's position directly.
	TeleportPlayer func(target *player.Player, x, y, z float64) error

	// NextEntityID allocates an ID shared with players and naturally spawned
	// mobs. It is supplied by the dispatcher for commands such as /summon.
	NextEntityID func() int32

	// FindPlayer resolves an online player across both Java and Bedrock
	// adapters. Commands that only need canonical player state should prefer it
	// over Manager, which contains Java network sessions only.
	FindPlayer func(name string) *player.Player

	// ListPlayers returns all online players from the edition-neutral game
	// registry. MaxPlayers is the configured server capacity.
	ListPlayers       func() []*player.Player
	MaxPlayers        int
	AvailableCommands []string

	// Reply sends command feedback to the issuing edition. SyncAbilities asks
	// that edition adapter to publish changed flight/permission state.
	Reply            func(text string) error
	SyncAbilities    func(*player.Player)
	DisconnectPlayer func(*player.Player, string) error
}

// CommandFunc is the handler signature for a built-in server command.
// A non-nil return value is formatted and sent to the issuing player as a
// system message; it is NOT logged as a server error.
type CommandFunc func(ctx CommandContext) error

type registeredCommand struct {
	fn           CommandFunc
	operatorOnly bool
	permission   string
	defaultAllow bool
}

type PermissionChecker func(player *player.Player, node string, defaultAllowed bool) bool

// ChatFormatter formats a single chat line.  prefix is the player's
// highest-weight group prefix (may be ""), username is the speaker, message is
// the raw chat text.
type ChatFormatter func(prefix, username, message string) string

// Dispatcher maps command names (lower-case) to their implementations.
// All methods are safe for concurrent use.
type Dispatcher struct {
	mu               sync.RWMutex
	cmds             map[string]registeredCommand
	nextEntityID     func() int32
	findPlayer       func(string) *player.Player
	listPlayers      func() []*player.Player
	teleportPlayer   func(*player.Player, float64, float64, float64) error
	disconnectPlayer func(*player.Player, string) error
	permission       PermissionChecker
	chatFormatter        ChatFormatter
	bedrockChatFormatter ChatFormatter
	groupPrefix          func(username string) string
	maxPlayers       int
}

// NewDispatcher returns an empty, ready-to-use Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{cmds: make(map[string]registeredCommand)}
}

// Register adds fn under the given name.  name is lowercased before storage
// and matched case-insensitively at dispatch time.
func (d *Dispatcher) Register(name string, fn CommandFunc) {
	d.mu.Lock()
	name = strings.ToLower(name)
	d.cmds[name] = registeredCommand{fn: fn, permission: commandPermissionNode(name), defaultAllow: true}
	d.mu.Unlock()
}

// RegisterOperator adds a command that may only be used by server operators.
func (d *Dispatcher) RegisterOperator(name string, fn CommandFunc) {
	d.mu.Lock()
	name = strings.ToLower(name)
	d.cmds[name] = registeredCommand{fn: fn, operatorOnly: true, permission: commandPermissionNode(name)}
	d.mu.Unlock()
}

// RequireOperator upgrades already-registered commands to operator-only.
func (d *Dispatcher) RequireOperator(names ...string) {
	d.mu.Lock()
	for _, name := range names {
		key := strings.ToLower(name)
		command, ok := d.cmds[key]
		if ok {
			command.operatorOnly = true
			command.defaultAllow = false
			d.cmds[key] = command
		}
	}
	d.mu.Unlock()
}

func commandPermissionNode(name string) string {
	return "gocraft.command." + strings.ToLower(strings.TrimSpace(name))
}

func (d *Dispatcher) SetPermissionChecker(check PermissionChecker) {
	d.mu.Lock()
	d.permission = check
	d.mu.Unlock()
}

// SetChatFormatter installs the function used to build chat lines.
func (d *Dispatcher) SetChatFormatter(f ChatFormatter) {
	d.mu.Lock()
	d.chatFormatter = f
	d.mu.Unlock()
}

// SetBedrockChatFormatter installs a separate formatter for Bedrock chat lines.
// It should produce text with only basic §-color codes (no hex, no gradients)
// since Bedrock clients do not support the §x hex sequence in chat.
func (d *Dispatcher) SetBedrockChatFormatter(f ChatFormatter) {
	d.mu.Lock()
	d.bedrockChatFormatter = f
	d.mu.Unlock()
}

// FormatBedrockChat is like FormatChat but uses the Bedrock-specific formatter.
// Falls back to FormatChat when no Bedrock formatter is set.
func (d *Dispatcher) FormatBedrockChat(username, message string) string {
	d.mu.RLock()
	f := d.bedrockChatFormatter
	gpfn := d.groupPrefix
	d.mu.RUnlock()

	if f == nil {
		return d.FormatChat(username, message)
	}
	prefix := ""
	if gpfn != nil {
		prefix = gpfn(username)
	}
	return f(prefix, username, message)
}

// SetGroupPrefixResolver installs the function used to look up a player's
// group prefix from the permission manager.
func (d *Dispatcher) SetGroupPrefixResolver(fn func(username string) string) {
	d.mu.Lock()
	d.groupPrefix = fn
	d.mu.Unlock()
}

// FormatChat produces the full chat line for the given player and message.
// It resolves the group prefix via the installed resolver, then calls the
// ChatFormatter.  Falls back to "<username> message" when nothing is set.
func (d *Dispatcher) FormatChat(username, message string) string {
	d.mu.RLock()
	f := d.chatFormatter
	gpfn := d.groupPrefix
	d.mu.RUnlock()

	prefix := ""
	if gpfn != nil {
		prefix = gpfn(username)
	}
	if f != nil {
		return f(prefix, username, message)
	}
	if prefix != "" {
		return prefix + "<" + username + "> " + message
	}
	return "<" + username + "> " + message
}

func (d *Dispatcher) CanUse(player *player.Player, name string) bool {
	d.mu.RLock()
	command, ok := d.cmds[strings.ToLower(name)]
	check := d.permission
	d.mu.RUnlock()
	if !ok {
		return false
	}
	if check != nil {
		return check(player, command.permission, command.defaultAllow)
	}
	return !command.operatorOnly && command.defaultAllow || player != nil && player.Operator
}

// SetEntityIDAllocator installs the game-wide allocator used by entity-spawning
// commands. The allocator may be configured once during server startup.
func (d *Dispatcher) SetEntityIDAllocator(allocate func() int32) {
	d.mu.Lock()
	d.nextEntityID = allocate
	d.mu.Unlock()
}

// SetPlayerFinder installs the edition-neutral online-player lookup used by
// administrative commands such as /op and /god.
func (d *Dispatcher) SetPlayerFinder(find func(string) *player.Player) {
	d.mu.Lock()
	d.findPlayer = find
	d.mu.Unlock()
}

// SetPlayerLister installs the edition-neutral player snapshot used by /list.
func (d *Dispatcher) SetPlayerLister(list func() []*player.Player) {
	d.mu.Lock()
	d.listPlayers = list
	d.mu.Unlock()
}

// SetPlayerTeleporter installs the edition-neutral target-side teleport
// bridge used by commands such as /tphere.
func (d *Dispatcher) SetPlayerTeleporter(teleport func(*player.Player, float64, float64, float64) error) {
	d.mu.Lock()
	d.teleportPlayer = teleport
	d.mu.Unlock()
}

func (d *Dispatcher) SetPlayerDisconnector(disconnect func(*player.Player, string) error) {
	d.mu.Lock()
	d.disconnectPlayer = disconnect
	d.mu.Unlock()
}

// SetMaxPlayers publishes the configured player capacity to commands.
func (d *Dispatcher) SetMaxPlayers(maxPlayers int) {
	d.mu.Lock()
	d.maxPlayers = maxPlayers
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
	command, ok := d.cmds[name]
	allocateEntityID := d.nextEntityID
	findPlayer := d.findPlayer
	listPlayers := d.listPlayers
	teleportPlayer := d.teleportPlayer
	disconnectPlayer := d.disconnectPlayer
	maxPlayers := d.maxPlayers
	checkPermission := d.permission
	d.mu.RUnlock()
	ctx.NextEntityID = allocateEntityID
	ctx.FindPlayer = findPlayer
	ctx.ListPlayers = listPlayers
	ctx.TeleportPlayer = teleportPlayer
	ctx.DisconnectPlayer = disconnectPlayer
	ctx.MaxPlayers = maxPlayers
	ctx.AvailableCommands = d.VisibleCommands(ctx.Player)

	if !ok {
		if ctx.Reply != nil {
			_ = ctx.Reply(fmt.Sprintf(`Unknown command: /%s`, name))
			return
		}
		_ = sendSystemMessage(ctx.Conn, fmt.Sprintf("Unknown command: /%s", name))
		return
	}
	allowed := !command.operatorOnly && command.defaultAllow || ctx.Player != nil && ctx.Player.Operator
	if checkPermission != nil {
		allowed = checkPermission(ctx.Player, command.permission, command.defaultAllow)
	}
	if !allowed {
		_ = sendCommandMessage(ctx, `You do not have permission to use this command`)
		return
	}
	if err := command.fn(ctx); err != nil {
		if ctx.Reply != nil {
			_ = ctx.Reply(`Error: ` + err.Error())
			return
		}
		_ = sendSystemMessage(ctx.Conn, "Error: "+err.Error())
	}
}
