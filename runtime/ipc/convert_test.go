package ipc

import (
	"reflect"
	"strings"
	"testing"

	abi "GoCraft/abi/v1"
	wire "GoCraft/abi/v1/wire"
)

func TestValueSurvivesARoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value abi.Value
	}{
		{name: "bool", value: abi.Bool(true)},
		{name: "false is not empty", value: abi.Bool(false)},
		{name: "int64", value: abi.Int64(-42)},
		{name: "double", value: abi.Double(0.015625)},
		{name: "string", value: abi.String("shop")},
		{name: "empty string", value: abi.String("")},
		{name: "bytes", value: abi.Bytes([]byte{0, 1, 255})},
		{name: "empty list", value: abi.List()},
		{name: "flat list", value: abi.List(abi.Int64(1), abi.String("two"))},
		{name: "nested list", value: abi.List(abi.List(abi.Bool(true), abi.List(abi.Int64(9))))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := encodeValue(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := decodeValue(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded, tc.value) {
				t.Fatalf("round trip = %+v, want %+v", decoded, tc.value)
			}
		})
	}
}

func TestEncodeValueRejectsAnUnsetKind(t *testing.T) {
	if _, err := encodeValue(abi.Value{}); err == nil {
		t.Fatal("encodeValue() accepted a value with no kind")
	}
}

func TestDecodeValueRejectsAnEmptyMessage(t *testing.T) {
	if _, err := decodeValue(&wire.Value{}); err == nil {
		t.Fatal("decodeValue() accepted a message with no kind set")
	}
	if _, err := decodeValue(nil); err == nil {
		t.Fatal("decodeValue() accepted nil")
	}
}

// A runtime is another process, so its output is untrusted: nesting has to be
// bounded on the way in rather than recursed into until the stack gives out.
func TestValueNestingIsBounded(t *testing.T) {
	deep := abi.Int64(1)
	for range maximumValueDepth + 1 {
		deep = abi.List(deep)
	}
	if _, err := encodeValue(deep); err == nil || !strings.Contains(err.Error(), "nested deeper") {
		t.Fatalf("encodeValue() error = %v", err)
	}

	message := &wire.Value{Kind: &wire.Value_Int64Value{Int64Value: 1}}
	for range maximumValueDepth + 1 {
		message = &wire.Value{Kind: &wire.Value_ListValue{ListValue: &wire.ValueList{Values: []*wire.Value{message}}}}
	}
	if _, err := decodeValue(message); err == nil || !strings.Contains(err.Error(), "nested deeper") {
		t.Fatalf("decodeValue() error = %v", err)
	}
}

func TestEventSurvivesARoundTrip(t *testing.T) {
	event := &abi.Event{
		Type:      "block.break",
		TypeID:    0,
		OnFailure: abi.FailureDeny,
		Fields: []abi.Value{
			abi.List(abi.Bytes([]byte("uuid")), abi.String("Romuald"), abi.String("java")),
			abi.List(abi.Int64(10), abi.Int64(64), abi.Int64(-3)),
			abi.String("minecraft:stone"),
		},
	}
	encoded, err := encodeEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeEvent(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, event) {
		t.Fatalf("round trip = %+v, want %+v", decoded, event)
	}
}

// The Go constants and the schema numbers deliberately differ: proto3 reserves
// zero for UNSPECIFIED. The mapping is this package's job and nothing else's.
func TestFailurePolicyIsRemappedNotReinterpreted(t *testing.T) {
	if abi.FailureAllow != 0 || abi.FailureDeny != 1 {
		t.Fatalf("domain policy numbering changed: allow=%d deny=%d", abi.FailureAllow, abi.FailureDeny)
	}
	allow, err := encodeFailurePolicy(abi.FailureAllow)
	if err != nil {
		t.Fatal(err)
	}
	if allow != wire.FailurePolicy_FAILURE_POLICY_ALLOW {
		t.Fatalf("allow encoded as %v", allow)
	}
	if _, err := decodeFailurePolicy(wire.FailurePolicy_FAILURE_POLICY_UNSPECIFIED); err == nil {
		t.Fatal("decodeFailurePolicy() accepted UNSPECIFIED")
	}
}

func TestVerdictSurvivesARoundTrip(t *testing.T) {
	verdict := abi.Verdict{
		Cancelled: true,
		Mutations: []abi.Mutation{
			{Path: []uint32{2, 0}, Value: abi.Int64(11)},
			{Path: nil, Value: abi.String("replaced")},
		},
		Effects: []abi.HostCall{
			{Type: "player.message", Fields: []abi.Value{abi.String("denied")}},
			{Type: "world.teleport"},
		},
	}
	encoded, err := encodeVerdict(verdict)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeVerdict(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, verdict) {
		t.Fatalf("round trip = %+v, want %+v", decoded, verdict)
	}
}

func TestEmptyVerdictSurvivesARoundTrip(t *testing.T) {
	encoded, err := encodeVerdict(abi.Verdict{})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeVerdict(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, abi.Verdict{}) {
		t.Fatalf("round trip = %+v, want an empty verdict", decoded)
	}
}