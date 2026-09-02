package bedrock

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"

	"GoCraft/core/game"
	"GoCraft/core/player"
	"github.com/GoCraft-MC/gocraft-abi/command"
)

func decimalBound(value float64) *float64 { return &value }

// shopTree covers each shape the renderer has to flatten: a command that runs
// alone and with a target, a literal branch, a typed argument and free text.
func shopTree() command.Root {
	return command.Root{Children: []command.Node{
		command.Literal{Name: "kill", Exec: 1, Children: []command.Node{
			command.Argument{Name: "target", Type: command.ArgPlayer, Exec: 1},
		}},
		command.Literal{Name: "gamemode", Children: []command.Node{
			command.Literal{Name: "creative", Exec: 2},
			command.Literal{Name: "survival", Exec: 2},
		}},
		command.Literal{Name: "shop", Children: []command.Node{
			command.Literal{Name: "sell", Children: []command.Node{
				command.Argument{
					Name: "price", Type: command.ArgDecimal,
					DecimalMin: decimalBound(0.01), Exec: 3,
				},
			}},
		}},
		command.Literal{Name: "say", Children: []command.Node{
			command.Argument{Name: "arguments", Type: command.ArgGreedy, Exec: 4},
		}},
	}}
}

func TestAvailableCommandsFlattensTheTree(t *testing.T) {
	pk := availableCommands(shopTree())
	if pk == nil {
		t.Fatal("a tree with commands rendered nothing")
	}
	byName := make(map[string]protocol.Command, len(pk.Commands))
	for _, entry := range pk.Commands {
		byName[entry.Name] = entry
	}

	// /kill runs alone and on a target: two signatures, because Bedrock has no
	// notion of an optional tail.
	kill, ok := byName["kill"]
	if !ok {
		t.Fatal("/kill is missing")
	}
	if len(kill.Overloads) != 2 {
		t.Fatalf("/kill has %d overloads, want 2", len(kill.Overloads))
	}
	if len(kill.Overloads[0].Parameters) != 0 {
		t.Fatal("/kill lost its bare form")
	}
	target := kill.Overloads[1].Parameters[0]
	if target.Name != "target" || target.Type != protocol.CommandArgValid|protocol.CommandArgTypeTarget {
		t.Fatalf("/kill target = %+v", target)
	}

	// A command that never runs on its own has no empty overload.
	gamemode := byName["gamemode"]
	if len(gamemode.Overloads) != 2 {
		t.Fatalf("/gamemode has %d overloads, want 2", len(gamemode.Overloads))
	}
	for _, overload := range gamemode.Overloads {
		if len(overload.Parameters) != 1 {
			t.Fatalf("/gamemode overload = %+v", overload)
		}
		if overload.Parameters[0].Type&protocol.CommandArgEnum == 0 {
			t.Fatal("/gamemode mode is not an enum")
		}
	}

	// A nested path becomes one signature holding every step.
	shop := byName["shop"]
	if len(shop.Overloads) != 1 || len(shop.Overloads[0].Parameters) != 2 {
		t.Fatalf("/shop overloads = %+v", shop.Overloads)
	}
	price := shop.Overloads[0].Parameters[1]
	if price.Type != protocol.CommandArgValid|protocol.CommandArgTypeFloat {
		t.Fatalf("/shop price type = %#x", price.Type)
	}

	say := byName["say"]
	if say.Overloads[0].Parameters[0].Type != protocol.CommandArgValid|protocol.CommandArgTypeMessage {
		t.Fatal("/say free text is not a message")
	}
}

// Every enum a parameter points at has to exist in the packet's table, or the
// client reads an index into nothing.
func TestAvailableCommandsResolvesEveryEnumIndex(t *testing.T) {
	pk := availableCommands(shopTree())
	for _, entry := range pk.Commands {
		for _, overload := range entry.Overloads {
			for _, parameter := range overload.Parameters {
				if parameter.Type&protocol.CommandArgEnum == 0 {
					continue
				}
				index := parameter.Type &^ (protocol.CommandArgValid | protocol.CommandArgEnum)
				if index >= uint32(len(pk.Enums)) {
					t.Fatalf("/%s %s points at enum %d of %d",
						entry.Name, parameter.Name, index, len(pk.Enums))
				}
				for _, value := range pk.Enums[index].ValueIndices {
					if value >= uint32(len(pk.EnumValues)) {
						t.Fatalf("enum %d points at value %d of %d",
							index, value, len(pk.EnumValues))
					}
				}
			}
		}
	}
}

// One word shared by two commands is stored once. The packet is large enough
// that the protocol warns against resending it.
func TestAvailableCommandsInternsEnumValues(t *testing.T) {
	root := command.Root{Children: []command.Node{
		command.Literal{Name: "gamemode", Children: []command.Node{command.Literal{Name: "creative", Exec: 1}}},
		command.Literal{Name: "gm", Children: []command.Node{command.Literal{Name: "creative", Exec: 2}}},
	}}
	pk := availableCommands(root)
	seen := 0
	for _, value := range pk.EnumValues {
		if value == "creative" {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("creative appears %d times in the value table", seen)
	}
}

// An empty tree sends nothing: AvailableCommands replaces the client's whole
// list, so an empty packet would take away what an earlier one gave.
func TestAvailableCommandsSkipsAnEmptyTree(t *testing.T) {
	if pk := availableCommands(command.Root{}); pk != nil {
		t.Fatalf("an empty tree rendered %+v", pk)
	}
}

// A tree deep in literals fans out, and the packet has to stay bounded.
func TestAvailableCommandsBoundsTheFanOut(t *testing.T) {
	children := make([]command.Node, 0, maximumOverloads*2)
	for index := 0; index < maximumOverloads*2; index++ {
		children = append(children, command.Literal{Name: string(rune('a'+index%26)) + string(rune('a'+index/26)), Exec: 1})
	}
	root := command.Root{Children: []command.Node{command.Literal{Name: "summon", Children: children}}}
	pk := availableCommands(root)
	if got := len(pk.Commands[0].Overloads); got > maximumOverloads {
		t.Fatalf("/summon rendered %d overloads", got)
	}
}

// recordingWriter stands in for a client connection, so what a send actually
// puts on the wire can be asserted without one.
type recordingWriter struct{ sent []packet.Packet }

func (w *recordingWriter) WritePacket(pk packet.Packet) error {
	w.sent = append(w.sent, pk)
	return nil
}

func listenerWithTree(t *testing.T, uuid [16]byte) (*Listener, *player.Player) {
	t.Helper()
	g := game.New()
	p := player.New(uuid, "joiner", player.ClientEditionBedrock)
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	l := &Listener{game: g}
	l.SetCommandTree(func(*player.Player) command.Root { return shopTree() })
	return l, p
}

// A joining player is sent their commands before addSession publishes the
// roster, so a send that resolves the session by uuid finds nothing and the
// player spends the session with an empty command list. The login path takes
// the connection it already holds; this is what says so.
func TestCommandsReachAPlayerNotYetInTheRoster(t *testing.T) {
	l, p := listenerWithTree(t, [16]byte{7})
	if l.sessionForPlayer(p.UUID) != nil {
		t.Fatal("the roster is not supposed to hold this player yet")
	}

	conn := &recordingWriter{}
	l.sendCommandsTo(conn, p.UUID)

	if len(conn.sent) != 1 {
		t.Fatalf("wrote %d packets, want 1", len(conn.sent))
	}
	if _, ok := conn.sent[0].(*packet.AvailableCommands); !ok {
		t.Fatalf("wrote %T, want *packet.AvailableCommands", conn.sent[0])
	}
}

// Nothing installed means nothing advertised: a server that never calls
// SetCommandTree behaves as it did before this edition rendered commands.
func TestNoCommandsSentWithoutATree(t *testing.T) {
	g := game.New()
	p := player.New([16]byte{8}, "joiner", player.ClientEditionBedrock)
	if err := g.AddPlayer(p); err != nil {
		t.Fatal(err)
	}

	conn := &recordingWriter{}
	(&Listener{game: g}).sendCommandsTo(conn, p.UUID)

	if len(conn.sent) != 0 {
		t.Fatalf("wrote %d packets with no tree installed, want 0", len(conn.sent))
	}
}

// The tree is rendered per player, so a uuid the game no longer knows has
// nobody to render for.
func TestNoCommandsSentForAPlayerWhoLeft(t *testing.T) {
	l, _ := listenerWithTree(t, [16]byte{9})

	conn := &recordingWriter{}
	l.sendCommandsTo(conn, [16]byte{200})

	if len(conn.sent) != 0 {
		t.Fatalf("wrote %d packets for an absent player, want 0", len(conn.sent))
	}
}
