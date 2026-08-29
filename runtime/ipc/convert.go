// Package ipc carries the plugin ABI to and from an out-of-process runtime.
//
// Everything here is shared by every such runtime — framing, correlation by
// seq, liveness, and the conversion below. Only the part that knows how to
// start a particular interpreter belongs to a specific runtime package.
//
// The generated wire types stop at this boundary. The bus, the mutation queue
// and the in-process runtimes work on the compact domain types in abi/v1, which
// cost no allocation per value; a Lua handler must not pay for a protocol it
// never speaks.
package ipc

import (
	"fmt"

	abi "GoCraft/abi/v1"
	wire "GoCraft/abi/v1/wire"
)

// maximumValueDepth bounds nesting on the way in. A runtime is a separate
// process and its output is untrusted input: without a limit, a value nested a
// million deep would recurse until the stack gives out.
const maximumValueDepth = 32

func encodeFailurePolicy(policy abi.FailurePolicy) (wire.FailurePolicy, error) {
	switch policy {
	case abi.FailureAllow:
		return wire.FailurePolicy_FAILURE_POLICY_ALLOW, nil
	case abi.FailureDeny:
		return wire.FailurePolicy_FAILURE_POLICY_DENY, nil
	default:
		return 0, fmt.Errorf("ipc: unknown failure policy %d", policy)
	}
}

// decodeFailurePolicy refuses UNSPECIFIED rather than defaulting it. The policy
// decides whether a silent subscriber cancels an event, so guessing it wrong is
// a gameplay bug that never reports itself.
func decodeFailurePolicy(policy wire.FailurePolicy) (abi.FailurePolicy, error) {
	switch policy {
	case wire.FailurePolicy_FAILURE_POLICY_ALLOW:
		return abi.FailureAllow, nil
	case wire.FailurePolicy_FAILURE_POLICY_DENY:
		return abi.FailureDeny, nil
	default:
		return 0, fmt.Errorf("ipc: missing failure policy")
	}
}

func encodeValue(value abi.Value) (*wire.Value, error) {
	return encodeValueAt(value, 1)
}

func encodeValueAt(value abi.Value, depth int) (*wire.Value, error) {
	if depth > maximumValueDepth {
		return nil, fmt.Errorf("ipc: value nested deeper than %d", maximumValueDepth)
	}
	switch value.Kind {
	case abi.ValueBool:
		return &wire.Value{Kind: &wire.Value_BoolValue{BoolValue: value.Bool}}, nil
	case abi.ValueInt64:
		return &wire.Value{Kind: &wire.Value_Int64Value{Int64Value: value.Int64}}, nil
	case abi.ValueDouble:
		return &wire.Value{Kind: &wire.Value_DoubleValue{DoubleValue: value.Double}}, nil
	case abi.ValueString:
		return &wire.Value{Kind: &wire.Value_StringValue{StringValue: value.String}}, nil
	case abi.ValueBytes:
		return &wire.Value{Kind: &wire.Value_BytesValue{BytesValue: value.Bytes}}, nil
	case abi.ValueList:
		list := &wire.ValueList{Values: make([]*wire.Value, 0, len(value.List))}
		for _, item := range value.List {
			encoded, err := encodeValueAt(item, depth+1)
			if err != nil {
				return nil, err
			}
			list.Values = append(list.Values, encoded)
		}
		return &wire.Value{Kind: &wire.Value_ListValue{ListValue: list}}, nil
	default:
		return nil, fmt.Errorf("ipc: unknown value kind %d", value.Kind)
	}
}

func decodeValue(value *wire.Value) (abi.Value, error) {
	return decodeValueAt(value, 1)
}

func decodeValueAt(value *wire.Value, depth int) (abi.Value, error) {
	if depth > maximumValueDepth {
		return abi.Value{}, fmt.Errorf("ipc: value nested deeper than %d", maximumValueDepth)
	}
	if value == nil {
		return abi.Value{}, fmt.Errorf("ipc: missing value")
	}
	switch kind := value.GetKind().(type) {
	case *wire.Value_BoolValue:
		return abi.Bool(kind.BoolValue), nil
	case *wire.Value_Int64Value:
		return abi.Int64(kind.Int64Value), nil
	case *wire.Value_DoubleValue:
		return abi.Double(kind.DoubleValue), nil
	case *wire.Value_StringValue:
		return abi.String(kind.StringValue), nil
	case *wire.Value_BytesValue:
		return abi.Bytes(kind.BytesValue), nil
	case *wire.Value_ListValue:
		items := make([]abi.Value, 0, len(kind.ListValue.GetValues()))
		for _, item := range kind.ListValue.GetValues() {
			decoded, err := decodeValueAt(item, depth+1)
			if err != nil {
				return abi.Value{}, err
			}
			items = append(items, decoded)
		}
		return abi.List(items...), nil
	default:
		// A value with no kind set is not an empty value: it means the sender
		// built a message it never filled in.
		return abi.Value{}, fmt.Errorf("ipc: value has no kind")
	}
}

func encodeValues(values []abi.Value) ([]*wire.Value, error) {
	encoded := make([]*wire.Value, 0, len(values))
	for _, value := range values {
		item, err := encodeValue(value)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, item)
	}
	return encoded, nil
}

func decodeValues(values []*wire.Value) ([]abi.Value, error) {
	if len(values) == 0 {
		return nil, nil
	}
	decoded := make([]abi.Value, 0, len(values))
	for _, value := range values {
		item, err := decodeValue(value)
		if err != nil {
			return nil, err
		}
		decoded = append(decoded, item)
	}
	return decoded, nil
}

func encodeEvent(event *abi.Event) (*wire.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("ipc: missing event")
	}
	fields, err := encodeValues(event.Fields)
	if err != nil {
		return nil, err
	}
	policy, err := encodeFailurePolicy(event.OnFailure)
	if err != nil {
		return nil, err
	}
	return &wire.Event{
		Type:      event.Type,
		TypeId:    event.TypeID,
		Fields:    fields,
		OnFailure: policy,
	}, nil
}

func decodeEvent(event *wire.Event) (*abi.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("ipc: missing event")
	}
	fields, err := decodeValues(event.GetFields())
	if err != nil {
		return nil, err
	}
	policy, err := decodeFailurePolicy(event.GetOnFailure())
	if err != nil {
		return nil, err
	}
	return &abi.Event{
		Type:      event.GetType(),
		TypeID:    event.GetTypeId(),
		Fields:    fields,
		OnFailure: policy,
	}, nil
}

func encodeHostCall(call abi.HostCall) (*wire.HostCall, error) {
	fields, err := encodeValues(call.Fields)
	if err != nil {
		return nil, err
	}
	return &wire.HostCall{Type: call.Type, Fields: fields}, nil
}

func decodeHostCall(call *wire.HostCall) (abi.HostCall, error) {
	if call == nil {
		return abi.HostCall{}, fmt.Errorf("ipc: missing host call")
	}
	fields, err := decodeValues(call.GetFields())
	if err != nil {
		return abi.HostCall{}, err
	}
	return abi.HostCall{Type: call.GetType(), Fields: fields}, nil
}

func encodeVerdict(verdict abi.Verdict) (*wire.Verdict, error) {
	mutations := make([]*wire.Mutation, 0, len(verdict.Mutations))
	for _, mutation := range verdict.Mutations {
		value, err := encodeValue(mutation.Value)
		if err != nil {
			return nil, err
		}
		mutations = append(mutations, &wire.Mutation{Path: mutation.Path, Value: value})
	}
	effects := make([]*wire.HostCall, 0, len(verdict.Effects))
	for _, effect := range verdict.Effects {
		call, err := encodeHostCall(effect)
		if err != nil {
			return nil, err
		}
		effects = append(effects, call)
	}
	return &wire.Verdict{Cancelled: verdict.Cancelled, Mutations: mutations, Effects: effects}, nil
}

func decodeVerdict(verdict *wire.Verdict) (abi.Verdict, error) {
	if verdict == nil {
		return abi.Verdict{}, fmt.Errorf("ipc: missing verdict")
	}
	var mutations []abi.Mutation
	for _, mutation := range verdict.GetMutations() {
		value, err := decodeValue(mutation.GetValue())
		if err != nil {
			return abi.Verdict{}, err
		}
		mutations = append(mutations, abi.Mutation{Path: mutation.GetPath(), Value: value})
	}
	var effects []abi.HostCall
	for _, effect := range verdict.GetEffects() {
		call, err := decodeHostCall(effect)
		if err != nil {
			return abi.Verdict{}, err
		}
		effects = append(effects, call)
	}
	return abi.Verdict{Cancelled: verdict.GetCancelled(), Mutations: mutations, Effects: effects}, nil
}