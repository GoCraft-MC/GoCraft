package command

import (
	"math"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"

	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"
)

// The payloads below are assembled byte by byte rather than with the generated
// marshaller: decoding must be verified against the wire format itself, not
// against a round trip through the encoder that shares its assumptions.
const (
	wireNodeLiteral  = uint64(wire.CommandNodeKind_COMMAND_NODE_KIND_LITERAL)
	wireNodeArgument = uint64(wire.CommandNodeKind_COMMAND_NODE_KIND_ARGUMENT)
)

func appendVarintField(data []byte, field protowire.Number, value uint64) []byte {
	data = protowire.AppendTag(data, field, protowire.VarintType)
	return protowire.AppendVarint(data, value)
}

func appendStringField(data []byte, field protowire.Number, value string) []byte {
	data = protowire.AppendTag(data, field, protowire.BytesType)
	return protowire.AppendString(data, value)
}

func appendMessageField(data []byte, field protowire.Number, value []byte) []byte {
	data = protowire.AppendTag(data, field, protowire.BytesType)
	return protowire.AppendBytes(data, value)
}

func validWireTree() []byte {
	argument := appendVarintField(nil, 1, wireNodeArgument)
	argument = appendStringField(argument, 2, "price")
	argument = appendVarintField(argument, 4, uint64(ArgDecimal))
	argument = appendVarintField(argument, 6, 7)
	argument = protowire.AppendTag(argument, 11, protowire.Fixed64Type)
	argument = protowire.AppendFixed64(argument, math.Float64bits(0.01))
	literal := appendVarintField(nil, 1, wireNodeLiteral)
	literal = appendStringField(literal, 2, "shop")
	literal = appendStringField(literal, 3, "shop.use")
	literal = appendMessageField(literal, 7, argument)
	tree := appendVarintField(nil, 1, CommandWireVersion)
	return appendMessageField(tree, 2, literal)
}

func TestDecodeTreeReadsTypedCommandNodes(t *testing.T) {
	data := appendStringField(validWireTree(), 99, "future field")
	root, err := DecodeTree(data)
	if err != nil {
		t.Fatal(err)
	}
	literal := root.Children[0].(Literal)
	if literal.Name != "shop" || literal.Permission != "shop.use" {
		t.Fatalf("literal = %+v", literal)
	}
	argument := literal.Children[0].(Argument)
	if argument.Name != "price" || argument.Type != ArgDecimal || argument.Exec != 7 {
		t.Fatalf("argument = %+v", argument)
	}
	if argument.DecimalMin == nil || *argument.DecimalMin != 0.01 {
		t.Fatalf("decimal minimum = %v", argument.DecimalMin)
	}
}

func TestDecodeTreeRejectsInvalidWirePayloads(t *testing.T) {
	literalWithType := appendVarintField(nil, 1, wireNodeLiteral)
	literalWithType = appendStringField(literalWithType, 2, "bad")
	literalWithType = appendVarintField(literalWithType, 4, uint64(ArgString))
	literalWithType = appendVarintField(literalWithType, 6, 1)
	badLiteral := appendVarintField(nil, 1, CommandWireVersion)
	badLiteral = appendMessageField(badLiteral, 2, literalWithType)
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{name: "missing version", data: validWireTree()[2:], want: "wire version"},
		{name: "future version", data: appendVarintField(nil, 1, 2), want: "unsupported"},
		{name: "truncated", data: []byte{0x12, 0x02, 0x08}, want: "cannot parse"},
		{name: "literal argument fields", data: badLiteral, want: "contains argument fields"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeTree(tc.data)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("DecodeTree() error = %v, want %q", err, tc.want)
			}
		})
	}
}
