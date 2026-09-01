package command

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"
)

// EncodeTree turns a validated tree into the commands.pb a bundle ships.
//
// It is the inverse of DecodeTree and lives beside it so the two cannot drift
// about what a node carries. Whoever builds a bundle encodes here and the host
// decodes there; a second encoder written into a build tool would be a second
// opinion on the format the host is the authority on.
//
// The tree is validated first, so a bundle cannot ship something the host will
// refuse at load. Finding that out at build time is the whole point of the
// build reading the same code the server does.
func EncodeTree(root Root) ([]byte, error) {
	if err := Validate(&root); err != nil {
		return nil, err
	}
	tree := &wire.CommandTree{Version: CommandWireVersion}
	for _, child := range root.Children {
		node, err := encodeNode(child)
		if err != nil {
			return nil, err
		}
		tree.Children = append(tree.Children, node)
	}
	encoded, err := proto.Marshal(tree)
	if err != nil {
		return nil, fmt.Errorf("command tree: %w", err)
	}
	return encoded, nil
}

func encodeNode(node Node) (*wire.CommandNode, error) {
	switch typed := node.(type) {
	case Literal:
		encoded := &wire.CommandNode{
			Kind:       wire.CommandNodeKind_COMMAND_NODE_KIND_LITERAL,
			Name:       typed.Name,
			Permission: typed.Permission,
			Executor:   uint32(typed.Exec),
		}
		children, err := encodeChildren(typed.Children)
		if err != nil {
			return nil, err
		}
		encoded.Children = children
		return encoded, nil
	case Argument:
		encoded := &wire.CommandNode{
			Kind:         wire.CommandNodeKind_COMMAND_NODE_KIND_ARGUMENT,
			Name:         typed.Name,
			ArgumentType: wire.CommandArgumentType(typed.Type),
			EnumValues:   typed.Enum,
			CustomType:   typed.CustomType,
			Executor:     uint32(typed.Exec),
			IntegerMin:   typed.IntegerMin,
			IntegerMax:   typed.IntegerMax,
			DecimalMin:   typed.DecimalMin,
			DecimalMax:   typed.DecimalMax,
		}
		children, err := encodeChildren(typed.Children)
		if err != nil {
			return nil, err
		}
		encoded.Children = children
		return encoded, nil
	}
	return nil, fmt.Errorf("command tree: cannot encode %T", node)
}

func encodeChildren(nodes []Node) ([]*wire.CommandNode, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	encoded := make([]*wire.CommandNode, 0, len(nodes))
	for _, node := range nodes {
		converted, err := encodeNode(node)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, converted)
	}
	return encoded, nil
}
