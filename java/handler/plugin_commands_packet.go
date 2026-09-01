package handler

import (
	"GoCraft/core/command"
	"GoCraft/java/protocol"
)

// Brigadier parser indices, continuing the registry the constants in
// commands_packet.go are drawn from. The order is the vanilla parser registry
// for this protocol, which is what makes 1 float, 2 double, 3 integer, 5 string
// and 14 item_stack the values already there.
const (
	parserBlockPos   int32 = 8
	parserBlockState int32 = 12
)

// String parser kinds, as the Commands packet encodes them.
const (
	stringSingleWord   int32 = 0
	stringGreedyPhrase int32 = 2
)

// Numeric bound flags. A parser sends only the bounds it has.
const (
	boundMinimum byte = 0x01
	boundMaximum byte = 0x02
)

// appendPluginCommands renders a plugin command tree into the Brigadier graph
// and reports the root children it produced.
//
// The tree is data, which is what makes this possible at all: the host holds
// the same structure a plugin's bundle shipped, so the client can be told about
// a command before the runtime hosting it has answered anything. §07 is where
// that pays off.
func appendPluginCommands(nodes *[]commandGraphNode, root command.Root) []int32 {
	return appendPluginChildren(nodes, root.Children)
}

func appendPluginChildren(nodes *[]commandGraphNode, children []command.Node) []int32 {
	if len(children) == 0 {
		return nil
	}
	indices := make([]int32, 0, len(children))
	for _, child := range children {
		if index, ok := appendPluginNode(nodes, child); ok {
			indices = append(indices, index)
		}
	}
	return indices
}

// appendPluginNode writes one node after its children, because a parent
// references them by index and an index only exists once the node does.
func appendPluginNode(nodes *[]commandGraphNode, node command.Node) (int32, bool) {
	graph := commandGraphNode{}
	var executor command.ExecID
	switch typed := node.(type) {
	case command.Literal:
		graph.flags = commandNodeLiteral
		graph.name = typed.Name
		graph.children = appendPluginChildren(nodes, typed.Children)
		executor = typed.Exec
	case command.Argument:
		graph.flags = commandNodeArgument
		graph.name = typed.Name
		graph.children = appendPluginChildren(nodes, typed.Children)
		graph.parser, graph.parserData = brigadierParser(typed)
		executor = typed.Exec
	default:
		return 0, false
	}
	if executor != 0 {
		graph.flags |= commandExecutable
	}
	*nodes = append(*nodes, graph)
	return int32(len(*nodes) - 1), true
}

// brigadierParser degrades an ArgType into what this client can render.
//
// The tree being data is what allows the degradation to live here rather than
// in every plugin: a type the protocol has no parser for becomes a single word
// the server resolves itself, and the plugin never learns that its argument
// arrived by a different route on one edition than on another. §07 calls this
// resolving at the boundary, and it is the same move the block registry already
// makes for state ids.
//
// The word count each parser accepts is the part that has to be right. A client
// validates against this graph before sending, so an argument the server reads
// as three tokens and the client renders as one single word would have the
// client refuse a line the server would have accepted.
func brigadierParser(argument command.Argument) (int32, func(*protocol.Builder)) {
	switch argument.Type {
	case command.ArgInteger:
		return parserInteger, integerRangeParser(argument)
	case command.ArgDecimal:
		return parserDouble, decimalRangeParser(argument)
	case command.ArgGreedy:
		return parserString, stringParser(stringGreedyPhrase)
	case command.ArgPlayer:
		return parserGameProfile, nil
	case command.ArgBlockPos:
		return parserBlockPos, nil
	case command.ArgBlockState:
		return parserBlockState, nil
	case command.ArgItem:
		return parserItemStack, nil
	default:
		// String, enum, duration and custom. An enum's values and a custom
		// type's suggestions belong in a completion the host serves, not in the
		// parser: the parser only has to accept the word.
		return parserString, stringParser(stringSingleWord)
	}
}

func integerRangeParser(argument command.Argument) func(*protocol.Builder) {
	minimum, maximum := argument.IntegerMin, argument.IntegerMax
	return func(b *protocol.Builder) {
		b.Byte(numericFlags(minimum != nil, maximum != nil))
		if minimum != nil {
			b.Int(clampInt32(*minimum))
		}
		if maximum != nil {
			b.Int(clampInt32(*maximum))
		}
	}
}

func decimalRangeParser(argument command.Argument) func(*protocol.Builder) {
	minimum, maximum := argument.DecimalMin, argument.DecimalMax
	return func(b *protocol.Builder) {
		b.Byte(numericFlags(minimum != nil, maximum != nil))
		if minimum != nil {
			b.Double(*minimum)
		}
		if maximum != nil {
			b.Double(*maximum)
		}
	}
}

func numericFlags(hasMinimum, hasMaximum bool) byte {
	var flags byte
	if hasMinimum {
		flags |= boundMinimum
	}
	if hasMaximum {
		flags |= boundMaximum
	}
	return flags
}

// clampInt32 keeps a bound the IR expresses in 64 bits inside what the packet
// carries. A bound wider than the field is the same as no bound at all, so
// saturating loses nothing a player could notice.
func clampInt32(bound int64) int32 {
	switch {
	case bound > int64(^uint32(0)>>1):
		return int32(^uint32(0) >> 1)
	case bound < -int64(^uint32(0)>>1)-1:
		return -int32(^uint32(0)>>1) - 1
	default:
		return int32(bound)
	}
}
