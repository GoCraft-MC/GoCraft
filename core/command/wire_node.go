package command

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

func decodeNode(data []byte, depth int, count *int) (Node, error) {
	*count++
	if *count > maximumCommandNodes || depth > maximumCommandDepth {
		return nil, fmt.Errorf("command tree: size limit exceeded")
	}
	decoded := wireNode{}
	for len(data) != 0 {
		number, wireType, tagSize := protowire.ConsumeTag(data)
		if tagSize < 0 {
			return nil, protowire.ParseError(tagSize)
		}
		data = data[tagSize:]
		var err error
		switch number {
		case 1:
			decoded.kind, data, err = consumeWireVarint(data, wireType)
		case 2:
			decoded.name, data, err = consumeWireString(data, wireType)
		case 3:
			decoded.permission, data, err = consumeWireString(data, wireType)
		case 4:
			decoded.argumentType, data, err = consumeWireVarint(data, wireType)
		case 5:
			var value string
			value, data, err = consumeWireString(data, wireType)
			decoded.enum = append(decoded.enum, value)
		case 6:
			decoded.executor, data, err = consumeWireVarint(data, wireType)
		case 7:
			var encoded []byte
			encoded, data, err = consumeWireBytes(data, wireType)
			if err == nil {
				var child Node
				child, err = decodeNode(encoded, depth+1, count)
				decoded.children = append(decoded.children, child)
			}
		case 8:
			decoded.customType, data, err = consumeWireString(data, wireType)
		case 9, 10:
			var value uint64
			value, data, err = consumeWireVarint(data, wireType)
			integer := protowire.DecodeZigZag(value)
			if number == 9 {
				decoded.integerMin = &integer
			} else {
				decoded.integerMax = &integer
			}
		case 11, 12:
			var bits uint64
			bits, data, err = consumeWireDouble(data, wireType)
			decimal := math.Float64frombits(bits)
			if number == 11 {
				decoded.decimalMin = &decimal
			} else {
				decoded.decimalMax = &decimal
			}
		default:
			size := protowire.ConsumeFieldValue(number, wireType, data)
			if size < 0 {
				err = protowire.ParseError(size)
			} else {
				data = data[size:]
			}
		}
		if err != nil {
			return nil, fmt.Errorf("command node %q field %d: %w", decoded.name, number, err)
		}
	}
	return decoded.commandNode()
}
