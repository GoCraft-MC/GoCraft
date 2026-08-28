package abi

// HostCall is a protocol-neutral request for a mutation or side effect.
// Calls are queued by the host and applied by the simulation tick.
type HostCall struct {
	Type   string
	Fields []Value
}
