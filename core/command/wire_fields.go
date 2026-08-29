package command

import (
	"fmt"
	"unicode/utf8"

	"google.golang.org/protobuf/encoding/protowire"
)

func consumeWireVarint(data []byte, wireType protowire.Type) (uint64, []byte, error) {
	if wireType != protowire.VarintType {
		return 0, nil, fmt.Errorf("expected a varint field")
	}
	value, size := protowire.ConsumeVarint(data)
	if size < 0 {
		return 0, nil, protowire.ParseError(size)
	}
	return value, data[size:], nil
}

func consumeWireBytes(data []byte, wireType protowire.Type) ([]byte, []byte, error) {
	if wireType != protowire.BytesType {
		return nil, nil, fmt.Errorf("expected a bytes field")
	}
	value, size := protowire.ConsumeBytes(data)
	if size < 0 {
		return nil, nil, protowire.ParseError(size)
	}
	return value, data[size:], nil
}

func consumeWireString(data []byte, wireType protowire.Type) (string, []byte, error) {
	value, rest, err := consumeWireBytes(data, wireType)
	if err != nil {
		return "", nil, err
	}
	if !utf8.Valid(value) {
		return "", nil, fmt.Errorf("string field is not valid UTF-8")
	}
	return string(value), rest, nil
}

func consumeWireDouble(data []byte, wireType protowire.Type) (uint64, []byte, error) {
	if wireType != protowire.Fixed64Type {
		return 0, nil, fmt.Errorf("expected a fixed64 field")
	}
	value, size := protowire.ConsumeFixed64(data)
	if size < 0 {
		return 0, nil, protowire.ParseError(size)
	}
	return value, data[size:], nil
}
