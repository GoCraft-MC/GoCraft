package bedrock

import (
	"sort"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"

	"GoCraft/core/command"
	"GoCraft/core/player"
)

// Bedrock's view of the command tree.
//
// It is the second renderer §18 asks for, over the same neutral tree the Java
// adapter renders. Until now this edition was told about no commands at all —
// not the built-ins, not a plugin's — so a Bedrock player had to already know
// what to type. The tree being data is what makes one description enough for
// two editions that model commands nothing alike.
//
// Where Brigadier is a graph, Bedrock is a list of flat signatures per command:
// every path from a command to something runnable becomes one overload. A tree
// therefore fans out here, which is why the node count is capped below.

// maximumOverloads bounds what one command may fan out to.
//
// A deep tree of literals multiplies: /summon alone has a mob per child and a
// profession under one of them. The packet is already large enough that the
// protocol warns against resending it, and a client is no better served by two
// hundred spellings of one command than by the first few.
const maximumOverloads = 32

// commandPermissionAny is the level every command is sent at. The field no
// longer gates anything client-side — Bedrock stopped deciding what may run —
// and the tree handed to this function was already pruned to its recipient.
const commandPermissionAny byte = 0

// availableCommands renders one player's tree.
//
// Returns nil when the tree is empty, so a server with nothing to advertise
// sends nothing rather than an empty packet the client would take as "every
// command you knew about is gone".
func availableCommands(root command.Root) *packet.AvailableCommands {
	if len(root.Children) == 0 {
		return nil
	}
	builder := &commandBuilder{enums: map[string]uint32{}}
	pk := &packet.AvailableCommands{}
	for _, node := range root.Children {
		literal, ok := node.(command.Literal)
		if !ok {
			continue
		}
		overloads := builder.overloads(literal)
		if len(overloads) == 0 {
			continue
		}
		pk.Commands = append(pk.Commands, protocol.Command{
			Name:            literal.Name,
			Description:     literal.Name,
			PermissionLevel: commandPermissionAny,
			Overloads:       overloads,
		})
	}
	if len(pk.Commands) == 0 {
		return nil
	}
	pk.EnumValues, pk.Enums = builder.build()
	return pk
}

// commandBuilder collects the enums the parameters point at.
//
// Bedrock keeps every enum value once, in one table, and a parameter refers to
// an enum by index. Interning them here is what stops /gamemode and /gm from
// each shipping their own copy of the same four words.
type commandBuilder struct {
	values     []string
	valueIndex map[string]uint32
	enums      map[string]uint32
	ordered    []protocol.CommandEnum
}

// overloads walks one command into the flat signatures Bedrock expects.
func (b *commandBuilder) overloads(literal command.Literal) []protocol.CommandOverload {
	var overloads []protocol.CommandOverload
	if literal.Exec != 0 {
		// The command runs with nothing after it, which Bedrock spells as an
		// overload with no parameters.
		overloads = append(overloads, protocol.CommandOverload{})
	}
	b.walk(literal.Children, nil, &overloads)
	if len(overloads) > maximumOverloads {
		overloads = overloads[:maximumOverloads]
	}
	return overloads
}

func (b *commandBuilder) walk(nodes []command.Node, prefix []protocol.CommandParameter, out *[]protocol.CommandOverload) {
	if len(*out) >= maximumOverloads {
		return
	}
	for _, node := range nodes {
		parameter, children, executable, ok := b.parameter(node)
		if !ok {
			continue
		}
		path := append(append([]protocol.CommandParameter(nil), prefix...), parameter)
		if executable {
			*out = append(*out, protocol.CommandOverload{Parameters: path})
		}
		b.walk(children, path, out)
		if len(*out) >= maximumOverloads {
			return
		}
	}
}

// parameter turns one node into what Bedrock shows for it.
func (b *commandBuilder) parameter(node command.Node) (protocol.CommandParameter, []command.Node, bool, bool) {
	switch typed := node.(type) {
	case command.Literal:
		// A literal is an enum of exactly itself. That is how Bedrock renders a
		// fixed word: there is no literal node in its model.
		return protocol.CommandParameter{
			Name: typed.Name,
			Type: protocol.CommandArgValid | protocol.CommandArgEnum | b.enum(typed.Name, typed.Name),
		}, typed.Children, typed.Exec != 0, true
	case command.Argument:
		return protocol.CommandParameter{
			Name: typed.Name,
			Type: b.argumentType(typed),
		}, typed.Children, typed.Exec != 0, true
	}
	return protocol.CommandParameter{}, nil, false, false
}

// argumentType maps an ArgType onto the closest thing this edition renders.
//
// Degrading happens here rather than in a plugin, exactly as it does for Java:
// a type Bedrock cannot show becomes text the server resolves itself, and the
// plugin never learns its argument arrived by a different route on one edition
// than on the other.
func (b *commandBuilder) argumentType(argument command.Argument) uint32 {
	switch argument.Type {
	case command.ArgInteger:
		return protocol.CommandArgValid | protocol.CommandArgTypeInt
	case command.ArgDecimal:
		return protocol.CommandArgValid | protocol.CommandArgTypeFloat
	case command.ArgPlayer:
		return protocol.CommandArgValid | protocol.CommandArgTypeTarget
	case command.ArgBlockPos:
		return protocol.CommandArgValid | protocol.CommandArgTypeBlockPosition
	case command.ArgGreedy:
		return protocol.CommandArgValid | protocol.CommandArgTypeMessage
	case command.ArgEnum:
		if len(argument.Enum) != 0 {
			return protocol.CommandArgValid | protocol.CommandArgEnum |
				b.enum(argument.Name, argument.Enum...)
		}
	}
	return protocol.CommandArgValid | protocol.CommandArgTypeString
}

// enum interns one enum and reports its index.
func (b *commandBuilder) enum(name string, options ...string) uint32 {
	key := name + "\x00" + joinOptions(options)
	if index, seen := b.enums[key]; seen {
		return index
	}
	indices := make([]uint32, 0, len(options))
	for _, option := range options {
		indices = append(indices, b.value(option))
	}
	index := uint32(len(b.ordered))
	b.ordered = append(b.ordered, protocol.CommandEnum{Type: name, ValueIndices: indices})
	b.enums[key] = index
	return index
}

func (b *commandBuilder) value(option string) uint32 {
	if b.valueIndex == nil {
		b.valueIndex = map[string]uint32{}
	}
	if index, seen := b.valueIndex[option]; seen {
		return index
	}
	index := uint32(len(b.values))
	b.values = append(b.values, option)
	b.valueIndex[option] = index
	return index
}

func (b *commandBuilder) build() ([]string, []protocol.CommandEnum) {
	return b.values, b.ordered
}

func joinOptions(options []string) string {
	sorted := append([]string(nil), options...)
	sort.Strings(sorted)
	joined := ""
	for _, option := range sorted {
		joined += option + "\x00"
	}
	return joined
}

// SetCommandTree installs the source of the commands a player may use.
//
// Installing it is what makes this edition see any command at all, so a server
// that never calls it behaves exactly as before: nothing is advertised and
// nothing is sent.
func (l *Listener) SetCommandTree(tree func(*player.Player) command.Root) {
	l.commandMu.Lock()
	l.commandTree = tree
	l.commandMu.Unlock()
}

// SendCommands tells one player what they may run.
//
// A no-op when no tree is installed or the player has nothing they may use:
// AvailableCommands replaces the client's whole list rather than adding to it,
// so an empty packet would take away what an earlier one gave.
func (l *Listener) SendCommands(playerUUID [16]byte) {
	l.commandMu.RLock()
	tree := l.commandTree
	l.commandMu.RUnlock()
	if tree == nil {
		return
	}
	session := l.sessionForPlayer(playerUUID)
	if session == nil {
		return
	}
	target := l.game.GetPlayer(playerUUID)
	if target == nil {
		return
	}
	if pk := availableCommands(tree(target)); pk != nil {
		_ = session.conn.WritePacket(pk)
	}
}

// RefreshCommands resends the list to everyone connected.
//
// Called when the registry changes — a plugin loaded, a command was revoked —
// because a client is told the whole list once and has no way to ask again.
func (l *Listener) RefreshCommands() {
	l.sessionsMu.RLock()
	connected := make([][16]byte, 0, len(l.sessions))
	for playerUUID := range l.sessions {
		connected = append(connected, playerUUID)
	}
	l.sessionsMu.RUnlock()
	for _, playerUUID := range connected {
		l.SendCommands(playerUUID)
	}
}
