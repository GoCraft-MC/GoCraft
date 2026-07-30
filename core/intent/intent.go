// Package intent defines the message types that network adapter goroutines
// (Java, Bedrock) post to the core simulation loop.
//
// Architecture:
//
//	Java handler goroutine  ──┐
//	                          ├──> intent.Queue ──> core simulation tick
//	Bedrock handler goroutine ──┘
//
// Handlers must NOT mutate core/player or core/world state directly.
// They post an Intent; the simulation/tick goroutine applies it and then
// publishes immutable outbound events to all session writers.
//
// This invariant ensures that Java and Bedrock adapters cannot race on shared
// world state, and that the tick goroutine remains the sole writer of all
// mutable simulation state (matching the non-player-entity ownership model
// established in M11).
package intent

import "GoCraft/core/spatial"

// Intent is the sealed interface for all network → simulation messages.
type Intent interface{ intentKind() }

// JoinIntent is posted when a player successfully authenticates and wants to
// enter the world.  The simulation assigns an entity ID and notifies all
// session writers.
type JoinIntent struct {
	PlayerUUID [16]byte
	Username   string
	// Edition is "java" or "bedrock".
	Edition string
}

// MoveIntent carries a position/rotation update from a client.
// The simulation applies it to player.Position and broadcasts to all other
// sessions.
type MoveIntent struct {
	PlayerUUID [16]byte
	Position   spatial.Vec3
	Rotation   spatial.Rotation
	OnGround   bool
}

// ChatIntent carries a chat message from a client.
// The simulation broadcasts it to all sessions.
type ChatIntent struct {
	PlayerUUID [16]byte
	Message    string
}

// DisconnectIntent is posted when a client connection closes (cleanly or not).
// The simulation removes the player from the world and notifies other sessions.
type DisconnectIntent struct {
	PlayerUUID [16]byte
	Reason     string
}

// TeleportIntent requests an authoritative server-side teleport.
// The simulation moves the player and triggers chunk streaming if needed.
type TeleportIntent struct {
	PlayerUUID [16]byte
	X, Y, Z   float64
}

func (JoinIntent) intentKind()       {}
func (MoveIntent) intentKind()       {}
func (ChatIntent) intentKind()       {}
func (DisconnectIntent) intentKind() {}
func (TeleportIntent) intentKind()   {}

// Queue is a buffered channel through which adapter goroutines submit Intents
// to the simulation tick.
//
// Post blocks if the queue is full, providing natural back-pressure on
// runaway clients.  The simulation drains the queue each tick via Drain.
type Queue struct {
	ch chan Intent
}

// NewQueue creates a Queue with the given buffer capacity.
// A capacity of 256–1024 is typical for a server handling tens of players.
func NewQueue(capacity int) *Queue {
	return &Queue{ch: make(chan Intent, capacity)}
}

// Post submits an intent.  Blocks if the queue is full.
func (q *Queue) Post(i Intent) {
	q.ch <- i
}

// TryPost submits an intent without blocking.  Returns false if the queue is
// full (the intent is dropped).  Prefer Post for authoritative mutations;
// TryPost is appropriate for high-frequency movement updates where a single
// dropped frame is acceptable.
func (q *Queue) TryPost(i Intent) bool {
	select {
	case q.ch <- i:
		return true
	default:
		return false
	}
}

// Drain removes and returns all queued intents without blocking.
// Call once per simulation tick to collect pending mutations.
func (q *Queue) Drain() []Intent {
	n := len(q.ch)
	if n == 0 {
		return nil
	}
	out := make([]Intent, 0, n)
	for range n {
		out = append(out, <-q.ch)
	}
	return out
}
