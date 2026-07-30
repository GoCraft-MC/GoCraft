// Package session manages the set of active Java Edition play sessions.
// Each connected player in the Play state has one Session entry; adapters for
// other editions (Bedrock, future) will have their own equivalent registries.
package session

import (
	"sync"

	"GoCraft/core/player"
	"GoCraft/java/network"
	"GoCraft/java/protocol"
)

// Session ties a canonical player (from core/) to their live Java connection.
// It is created when the player enters the Play state and removed on disconnect.
type Session struct {
	Player *player.Player
	Conn   *network.ClientConn
}

// Manager is a concurrent-safe registry of active Java play sessions.
// All methods are safe for concurrent use across multiple goroutines.
type Manager struct {
	mu       sync.RWMutex
	sessions map[[16]byte]*Session
}

// NewManager returns an empty, ready-to-use Manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[[16]byte]*Session)}
}

// Add registers s in the manager. Replaces any existing session for the same UUID.
func (m *Manager) Add(s *Session) {
	m.mu.Lock()
	m.sessions[s.Player.UUID] = s
	m.mu.Unlock()
}

// Remove deregisters the session identified by uuid.
func (m *Manager) Remove(uuid [16]byte) {
	m.mu.Lock()
	delete(m.sessions, uuid)
	m.mu.Unlock()
}

// Get returns the session for uuid, or (nil, false) if not present.
func (m *Manager) Get(uuid [16]byte) (*Session, bool) {
	m.mu.RLock()
	s, ok := m.sessions[uuid]
	m.mu.RUnlock()
	return s, ok
}

// ForEachExcept calls fn for every session except the one with excludeUUID.
// The read lock is held for the duration; fn must not call Add or Remove.
func (m *Manager) ForEachExcept(excludeUUID [16]byte, fn func(*Session)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for uuid, s := range m.sessions {
		if uuid != excludeUUID {
			fn(s)
		}
	}
}

// ForEach calls fn for every session in the manager.
// The read lock is held for the duration; fn must not call Add or Remove.
func (m *Manager) ForEach(fn func(*Session)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		fn(s)
	}
}

// BroadcastExcept writes pkt to every session except excludeUUID.
// Write errors are silently ignored; each connection's own goroutine handles
// cleanup when the next read or write fails.
func (m *Manager) BroadcastExcept(excludeUUID [16]byte, pkt *protocol.Packet) {
	m.ForEachExcept(excludeUUID, func(s *Session) {
		_ = s.Conn.WritePacket(pkt)
	})
}
