package gocraft

// The contract every generated event satisfies.
//
// The events themselves live in events.gen.go, emitted from abi/v1/events.proto
// alongside the host emitters and the Java classes. What stays here is the part
// no schema describes: what it means to be an event at all.

// Event is anything the host can dispatch to a handler.
type Event interface {
	Type() string
}

// CancellableEvent is an event a handler may prevent.
//
// A separate interface rather than a method on every event, because an
// observational event simply does not offer it: the tick never waits for one,
// and a Cancel that silently did nothing would be worse than its absence.
type CancellableEvent interface {
	Event
	Cancel()
	Cancelled() bool
}

// EventHandler receives an event by name, for a type this build does not know.
type EventHandler func(Event)
