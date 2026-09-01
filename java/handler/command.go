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

	"GoCraft/core/command"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"
	"GoCraft/java/session"
)

// CommandContext carries every resource a command handler might need.
//
// It holds no connection, deliberately. A Bedrock player has no
// *network.ClientConn, so a handler that reached for one answered nobody on
// that edition — silently, because every Java send helper treats a nil
// connection as a no-op. Feedback and client resynchronisation therefore travel
// through the callbacks below, which Dispatch fills from bridges the server
// installs once and which know both editions.
//
// Manager is the exception that proves it: it holds Java network sessions only,
// so anything that merely needs canonical player state must use FindPlayer and
// ListPlayers instead.
type CommandContext struct {
	Player  *player.Player
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

	// Reply sends command feedback to the issuing player, whichever edition
	// they are on. ReplyTo does the same for anyone else online, which is what
	// commands acting on a target need — /heal notifies the healed player, and
	// that player may well be on the other edition.
	Reply   func(text string) error
	ReplyTo func(target *player.Player, text string) error

	// ReplyLink sends a clickable link where the edition supports one and
	// degrades to text where it does not, rather than leaving Bedrock players
	// with nothing at all.
	ReplyLink func(text, link string) error

	// SyncAbilities republishes the issuing player's local state — game mode,
	// flight, speeds — after a command changed it. Both adapters send their own
	// packets for this; neither is reachable from here.
	SyncAbilities func(*player.Player)

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

// PluginCommands runs one line against the commands plugins registered.
//
// It reports whether the line named one at all, which is what lets the two
// command systems live side by side while §18's extraction is unfinished: a
// line this returns false for was never a plugin's, and the dispatcher answers
// for it as it always did. A returned error is a sentence for whoever typed the
// line, not a server fault.
type PluginCommands func(sender *player.Player, line string) (handled bool, err error)

// ChatFormatter formats a single chat line.  prefix is the player's
// highest-weight group prefix (may be ""), username is the speaker, message is
// the raw chat text.
type ChatFormatter func(prefix, username, message string) string

// Dispatcher maps command names (lower-case) to their implementations.
// All methods are safe for concurrent use.
type Dispatcher struct {
	mu                   sync.RWMutex
	cmds                 map[string]registeredCommand
	nextEntityID         func() int32
	findPlayer           func(string) *player.Player
	listPlayers          func() []*player.Player
	teleportPlayer       func(*player.Player, float64, float64, float64) error
	disconnectPlayer     func(*player.Player, string) error
	permission           PermissionChecker
	pluginCommands       PluginCommands
	pluginTree           func(*player.Player) command.Root
	messenger            func(*player.Player, string) error
	linkMessenger        func(*player.Player, string, string) error
	syncAbilities        func(*player.Player)
	chatFormatter        ChatFormatter
	bedrockChatFormatter ChatFormatter
	groupPrefix          func(username string) string
	maxPlayers           int
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
		registered, ok := d.cmds[key]
		if ok {
			registered.operatorOnly = true
			registered.defaultAllow = false
			d.cmds[key] = registered
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

// SetPluginCommands installs the bridge to the plugin command registry.
//
// One hook rather than a merged table: the two registries validate, namespace
// and check permissions differently, and copying plugin commands in here would
// be a second place they are written down. Built-in names are refused to
// plugins at registration, so there is no precedence to invent — a line is one
// or the other, never both.
func (d *Dispatcher) SetPluginCommands(run PluginCommands) {
	d.mu.Lock()
	d.pluginCommands = run
	d.mu.Unlock()
}

// SetPluginCommandTree installs the source of the command tree sent to a
// client, pruned to what that client may use.
func (d *Dispatcher) SetPluginCommandTree(tree func(*player.Player) command.Root) {
	d.mu.Lock()
	d.pluginTree = tree
	d.mu.Unlock()
}

// PluginCommandTree returns the plugin commands to advertise to one player.
//
// Empty until plugins load, and empty forever on a server that has none, which
// is what keeps the graph a client receives identical to today's when nothing
// is installed.
func (d *Dispatcher) PluginCommandTree(p *player.Player) command.Root {
	d.mu.RLock()
	tree := d.pluginTree
	d.mu.RUnlock()
	if tree == nil {
		return command.Root{}
	}
	return tree(p)
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

// SetMessenger installs the edition-neutral bridge that delivers command
// feedback to any online player.
//
// It is what replaced the connection CommandContext used to carry. Only the
// server sees both adapters, so only the server can write this — which is also
// why it is a bridge rather than something java/handler resolves itself.
func (d *Dispatcher) SetMessenger(send func(*player.Player, string) error) {
	d.mu.Lock()
	d.messenger = send
	d.mu.Unlock()
}

// SetLinkMessenger installs the bridge for clickable links. A sender with no
// way to render one is expected to fall back to text rather than send nothing.
func (d *Dispatcher) SetLinkMessenger(send func(*player.Player, string, string) error) {
	d.mu.Lock()
	d.linkMessenger = send
	d.mu.Unlock()
}

// SetAbilitySync installs the bridge that republishes a player's game mode,
// flight and speeds after a command changed them.
func (d *Dispatcher) SetAbilitySync(sync func(*player.Player)) {
	d.mu.Lock()
	d.syncAbilities = sync
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
// Unknown commands and handler errors are reported to the issuing player as a
// system message.  Dispatch itself never returns an error to the caller because
// command failures are player-facing, not server-fatal.
//
// It is also where the context is completed: the caller supplies who is asking
// and what they asked, and the callbacks that reach either edition are filled
// in from the bridges installed on the dispatcher. A caller that already set
// one keeps it, which is what lets a test inject its own without a server.
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
	runPluginCommand := d.pluginCommands
	messenger := d.messenger
	linkMessenger := d.linkMessenger
	syncAbilities := d.syncAbilities
	d.mu.RUnlock()
	ctx.NextEntityID = allocateEntityID
	ctx.FindPlayer = findPlayer
	ctx.ListPlayers = listPlayers
	ctx.TeleportPlayer = teleportPlayer
	ctx.DisconnectPlayer = disconnectPlayer
	ctx.MaxPlayers = maxPlayers
	ctx.AvailableCommands = d.VisibleCommands(ctx.Player)
	fillFeedback(&ctx, messenger, linkMessenger, syncAbilities)

	if !ok {
		// Plugins are asked only once no built-in answers to the name. That
		// ordering costs nothing — the two sets cannot overlap — and it keeps
		// a plugin from being consulted on every /gamemode ever typed.
		if runPluginCommand != nil {
			if handled, err := runPluginCommand(ctx.Player, input); handled {
				if err != nil {
					_ = sendCommandMessage(ctx, err.Error())
				}
				return
			}
		}
		_ = sendCommandMessage(ctx, fmt.Sprintf("Unknown command: /%s", name))
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
		_ = sendCommandMessage(ctx, "Error: "+err.Error())
	}
}
