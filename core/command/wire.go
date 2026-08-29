package command

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	abi "GoCraft/abi/v1"
)

const CommandWireVersion = 1

// DecodeTree validates and decodes the generated commands.pb payload.
//
// Parsing is the generated codec's job; this package only turns the wire
// message into the neutral tree the rest of the server uses. Generated types
// never leave this file and wire_convert.go.
//
// Size is bounded before this point: a bundle entry is read under a 4 MiB cap,
// and the protobuf runtime refuses deeply nested messages on its own. The node
// and depth limits below are enforced again during conversion, because those
// are the server's limits rather than the format's.
func DecodeTree(data []byte) (Root, error) {
	var tree abi.CommandTree
	if err := proto.Unmarshal(data, &tree); err != nil {
		return Root{}, fmt.Errorf("command tree: %w", err)
	}
	if tree.GetVersion() != CommandWireVersion {
		return Root{}, fmt.Errorf("command tree: wire version %d is unsupported", tree.GetVersion())
	}
	root := Root{}
	nodeCount := 0
	for _, child := range tree.GetChildren() {
		node, err := convertNode(child, 1, &nodeCount)
		if err != nil {
			return Root{}, err
		}
		root.Children = append(root.Children, node)
	}
	if err := Validate(&root); err != nil {
		return Root{}, err
	}
	return root, nil
}