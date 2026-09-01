package handler

import (
	"testing"

	"GoCraft/core/command"
	"GoCraft/core/player"
)

func decimalBound(value float64) *float64 { return &value }

func integerBound(value int64) *int64 { return &value }

// pluginGraph is one command of every shape the encoder has to render: a
// literal branch, a bounded decimal, a player, a three-token position and an
// item.
func pluginGraph() command.Root {
	return command.Root{Children: []command.Node{command.Literal{
		Name: "shop", Children: []command.Node{
			command.Literal{Name: "sell", Children: []command.Node{
				command.Argument{
					Name: "price", Type: command.ArgDecimal,
					DecimalMin: decimalBound(0.01), DecimalMax: decimalBound(1000), Exec: 1,
				},
			}},
			command.Literal{Name: "give", Children: []command.Node{
				command.Argument{Name: "target", Type: command.ArgPlayer, Children: []command.Node{
					command.Argument{Name: "what", Type: command.ArgItem, Exec: 2},
				}},
			}},
			command.Literal{Name: "at", Children: []command.Node{
				command.Argument{Name: "where", Type: command.ArgBlockPos, Children: []command.Node{
					command.Argument{Name: "note", Type: command.ArgGreedy, Exec: 3},
				}},
			}},
			command.Argument{
				Name: "slot", Type: command.ArgInteger,
				IntegerMin: integerBound(1), IntegerMax: integerBound(9), Exec: 4,
			},
		},
	}}}
}

// childNamed finds a node's child by name, so a test reads as a path through
// the graph rather than as index arithmetic.
func childNamed(t *testing.T, nodes []commandTestNode, parent commandTestNode, name string) commandTestNode {
	t.Helper()
	for _, index := range parent.children {
		if nodes[index].name == name {
			return nodes[index]
		}
	}
	t.Fatalf("no child named %q", name)
	return commandTestNode{}
}

func TestCommandsPacketCarriesPluginCommands(t *testing.T) {
	nodes, rootIndex, err := parseCommandTestGraph(buildCommandsPacketFor(pluginGraph()).Data)
	if err != nil {
		t.Fatal(err)
	}
	root := nodes[rootIndex]

	shop := childNamed(t, nodes, root, "shop")
	if shop.flags&0x03 != 0x01 {
		t.Fatalf("shop is not a literal: flags %#x", shop.flags)
	}
	if shop.flags&0x04 != 0 {
		t.Fatal("a node with no executor was marked executable")
	}

	price := childNamed(t, nodes, childNamed(t, nodes, shop, "sell"), "price")
	if price.flags&0x03 != 0x02 {
		t.Fatalf("price is not an argument: flags %#x", price.flags)
	}
	if price.flags&0x04 == 0 {
		t.Fatal("the node carrying an executor was not marked executable")
	}

	// Three tokens on the server must be three tokens on the client, or the
	// client refuses a line the server would have accepted.
	where := childNamed(t, nodes, childNamed(t, nodes, shop, "at"), "where")
	if len(where.children) != 1 {
		t.Fatalf("block position has %d children", len(where.children))
	}
	childNamed(t, nodes, where, "note")

	target := childNamed(t, nodes, childNamed(t, nodes, shop, "give"), "target")
	childNamed(t, nodes, target, "what")
	childNamed(t, nodes, shop, "slot")
}

// A server with no plugins sends exactly the graph it always sent.
func TestCommandsPacketIsUnchangedWithoutPlugins(t *testing.T) {
	withoutPlugins := buildCommandsPacket()
	empty := buildCommandsPacketFor(command.Root{})
	if string(withoutPlugins.Data) != string(empty.Data) {
		t.Fatal("an empty plugin tree changed the command graph")
	}
}

// The tree a client is sent comes from the dispatcher, and a dispatcher nobody
// gave one to answers with an empty tree rather than nil-panicking.
func TestPluginCommandTreeDefaultsToEmpty(t *testing.T) {
	dispatcher := NewDispatcher()
	if got := dispatcher.PluginCommandTree(nil); len(got.Children) != 0 {
		t.Fatalf("unset tree = %v", got)
	}
	dispatcher.SetPluginCommandTree(func(*player.Player) command.Root { return pluginGraph() })
	if got := dispatcher.PluginCommandTree(nil); len(got.Children) != 1 {
		t.Fatalf("installed tree = %v", got)
	}
}
