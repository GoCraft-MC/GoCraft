package bedrock

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"

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
