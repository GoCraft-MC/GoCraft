// Package abi defines the language-neutral GoCraft plugin ABI.
package abi

// FailurePolicy decides a cancellable event when a subscriber fails to reply.
type FailurePolicy uint8

const (
	FailureAllow FailurePolicy = iota
	FailureDeny
)

// ValueKind identifies the active field in Value.
type ValueKind uint8

const (
	ValueInvalid ValueKind = iota
	ValueBool
	ValueInt64
	ValueDouble
	ValueString
	ValueBytes
	ValueList
)

// Value is the positional value carried by events, configs, and store records.
// Only the field selected by Kind is meaningful.
type Value struct {
	Kind   ValueKind
	Bool   bool
	Int64  int64
	Double float64
	String string
	Bytes  []byte
	List   []Value
}

// Event is the generic envelope used for native and plugin-defined events.
type Event struct {
	Type      string
	TypeID    uint32
	Fields    []Value
	OnFailure FailurePolicy
}

// Mutation replaces the value at a positional path after a subscriber runs.
type Mutation struct {
	Path  []uint32
	Value Value
}

// Verdict is returned by one subscriber.
type Verdict struct {
	Cancelled bool
	Mutations []Mutation
	Effects   []HostCall
}
