package command

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

const CommandWireVersion = 1

// DecodeTree validates and decodes the generated commands.pb payload.
func DecodeTree(data []byte) (Root, error) {
	var root Root
	version, versionSeen := uint64(0), false
	nodeCount := 0
	for len(data) != 0 {
		number, wireType, tagSize := protowire.ConsumeTag(data)
		if tagSize < 0 {
			return Root{}, protowire.ParseError(tagSize)
		}
		data = data[tagSize:]
		switch number {
		case 1:
			if wireType != protowire.VarintType || versionSeen {
				return Root{}, fmt.Errorf("command tree: invalid version field")
			}
			value, size := protowire.ConsumeVarint(data)
			if size < 0 {
				return Root{}, protowire.ParseError(size)
			}
			version, versionSeen, data = value, true, data[size:]
		case 2:
			if wireType != protowire.BytesType {
				return Root{}, fmt.Errorf("command tree: invalid child field")
			}
			encoded, size := protowire.ConsumeBytes(data)
			if size < 0 {
				return Root{}, protowire.ParseError(size)
			}
			node, err := decodeNode(encoded, 1, &nodeCount)
			if err != nil {
				return Root{}, err
			}
			root.Children = append(root.Children, node)
			data = data[size:]
		default:
			size := protowire.ConsumeFieldValue(number, wireType, data)
			if size < 0 {
				return Root{}, protowire.ParseError(size)
			}
			data = data[size:]
		}
	}
	if !versionSeen || version != CommandWireVersion {
		return Root{}, fmt.Errorf("command tree: wire version %d is unsupported", version)
	}
	if err := Validate(&root); err != nil {
		return Root{}, err
	}
	return root, nil
}
