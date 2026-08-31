package gocraft

const (
	EventBlockBreak = "block.break"
	EventPlayerJoin = "player.join"
)

// Event is a protocol-independent gameplay fact.
type Event interface {
	Type() string
}

// CancellableEvent is implemented by events emitted before a mutation.
type CancellableEvent interface {
	Event
	Cancel()
	Cancelled() bool
}

// EventHandler receives events serially for one plugin.
type EventHandler func(Event)

// PlayerJoinEvent is emitted after a player becomes reachable.
type PlayerJoinEvent struct {
	Player      Player
	Permissions map[string]bool
}

func (*PlayerJoinEvent) Type() string { return EventPlayerJoin }

// BlockBreakEvent is emitted before a block is removed.
type BlockBreakEvent struct {
	Player      Player
	Position    BlockPos
	Block       Block
	Tool        string
	Permissions map[string]bool
	cancelled   bool
}

func (*BlockBreakEvent) Type() string { return EventBlockBreak }

func (e *BlockBreakEvent) Cancel() { e.cancelled = true }

func (e *BlockBreakEvent) Cancelled() bool { return e.cancelled }
