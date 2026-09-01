package handler

// Commands packet builder for Minecraft Java Edition 1.21.4.
//
// The graph is rendered from the neutral command tree rather than built here.
// It used to be written by hand, beside a dispatcher that routed on names it
// knew nothing about, so the two could disagree about which commands existed —
// and a Bedrock client got neither, because a Brigadier graph is not something
// the other edition can read. §18 makes the tree the one description both
// editions render.

import (
	"GoCraft/core/command"
	"GoCraft/java/protocol"
)

const (
	commandNodeRoot     byte = 0x00
	commandNodeLiteral  byte = 0x01
	commandNodeArgument byte = 0x02
	commandExecutable   byte = 0x04

	parserDouble      int32 = 2
	parserInteger     int32 = 3
	parserString      int32 = 5
	parserGameProfile int32 = 7
	parserItemStack   int32 = 14
)

type commandGraphNode struct {
	flags      byte
	children   []int32
	name       string
	parser     int32
	parserData func(*protocol.Builder)
}

// buildCommandsPacket renders one tree, already pruned to its recipient.
//
// Pruning happens where permissions are understood — the registry snapshot —
// rather than through a filter passed down here. A node that reaches this
// function is one the player may use.
func buildCommandsPacket(root command.Root) *protocol.Packet {
	nodes := []commandGraphNode{{flags: commandNodeRoot}}
	nodes[0].children = appendCommands(&nodes, root)

	b := protocol.NewBuilder(packetIDCommands).VarInt(int32(len(nodes)))
	for _, node := range nodes {
		b.Byte(node.flags)
		nodeChildren(b, node.children...)
		switch node.flags & 0x03 {
		case commandNodeLiteral:
			b.String(node.name)
		case commandNodeArgument:
			b.String(node.name).VarInt(node.parser)
			if node.parserData != nil {
				node.parserData(b)
			}
		}
	}
	b.VarInt(0) // root node index
	return b.Build()
}

func stringParser(kind int32) func(*protocol.Builder) {
	return func(b *protocol.Builder) {
		b.VarInt(kind)
	}
}

func nodeChildren(b *protocol.Builder, indices ...int32) {
	b.VarInt(int32(len(indices)))
	for _, index := range indices {
		b.VarInt(index)
	}
}
