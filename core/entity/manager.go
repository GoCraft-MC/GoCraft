package entity

import "sync"

// Manager is a thread-safe registry of non-player entities.
// Entity IDs must be assigned by the caller before calling Add; the Manager
// does not allocate IDs (the game core's shared counter in core/game is the
// single source of entity IDs for both players and mobs).
type Manager struct {
	mu       sync.RWMutex
	entities map[int32]*Entity
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{entities: make(map[int32]*Entity)}
}

// Add inserts e into the registry.  Overwrites any existing entry with the
// same EntityID (callers should ensure IDs are unique).
func (m *Manager) Add(e *Entity) {
	m.mu.Lock()
	m.entities[e.EntityID] = e
	m.mu.Unlock()
}

// Remove deletes the entity with the given ID.  No-op if it does not exist.
func (m *Manager) Remove(id int32) {
	m.mu.Lock()
	delete(m.entities, id)
	m.mu.Unlock()
}

// Get returns the entity with the given ID and true, or nil and false if it
// does not exist.
func (m *Manager) Get(id int32) (*Entity, bool) {
	m.mu.RLock()
	e, ok := m.entities[id]
	m.mu.RUnlock()
	return e, ok
}

// Snapshot returns a slice of all currently registered entities.  The slice
// is safe to iterate without holding any lock; elements are live pointers so
// callers must not store the slice beyond the immediate iteration.
func (m *Manager) Snapshot() []*Entity {
	m.mu.RLock()
	out := make([]*Entity, 0, len(m.entities))
	for _, e := range m.entities {
		out = append(out, e)
	}
	m.mu.RUnlock()
	return out
}

// Count returns the number of entities currently registered.
func (m *Manager) Count() int {
	m.mu.RLock()
	n := len(m.entities)
	m.mu.RUnlock()
	return n
}
