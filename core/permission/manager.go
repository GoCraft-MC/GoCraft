package permission

import (
	"sync"
)

type Manager struct {
	mu       sync.RWMutex
	store    Store
	document Document
}

// Open loads a Document from store and returns a ready Manager.
func Open(store Store) (*Manager, error) {
	doc, err := store.Load()
	if err != nil {
		return nil, err
	}
	if err := validateDocument(doc); err != nil {
		return nil, err
	}
	return &Manager{store: store, document: doc}, nil
}

// Load is a convenience wrapper that opens a JSON file store at path.
// It is equivalent to Open(JSONStore(path)).
func Load(path string) (*Manager, error) {
	return Open(JSONStore(path))
}

// NewMemory returns a Manager that is pre-seeded with DefaultDocument and
// never persists changes. Intended for tests.
func NewMemory() *Manager {
	return &Manager{store: noopStore{}, document: DefaultDocument()}
}

func (m *Manager) Snapshot() Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneDocument(m.document)
}

func (m *Manager) Replace(document Document) error {
	if err := validateDocument(document); err != nil {
		return err
	}
	document = cloneDocument(document)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.store.Save(document); err != nil {
		return err
	}
	m.document = document
	return nil
}

// Reload re-reads the Document from the backing store and replaces the
// in-memory copy atomically.
func (m *Manager) Reload() error {
	doc, err := m.store.Load()
	if err != nil {
		return err
	}
	if err := validateDocument(doc); err != nil {
		return err
	}
	m.mu.Lock()
	m.document = cloneDocument(doc)
	m.mu.Unlock()
	return nil
}

// Close releases resources held by the backing store.
func (m *Manager) Close() error {
	return m.store.Close()
}

// GroupPrefix returns the chat prefix of the highest-weight group the player
// belongs to. Returns "" when no group has a prefix set.
func (m *Manager) GroupPrefix(username string) string {
	username = Normalize(username)
	m.mu.RLock()
	defer m.mu.RUnlock()

	user := m.document.Users[username]
	groups := append([]string{"default"}, user.Groups...)

	bestPrefix := ""
	bestWeight := -1
	for _, name := range groups {
		if g, ok := m.document.Groups[Normalize(name)]; ok && g.Prefix != "" {
			if g.Weight > bestWeight {
				bestPrefix = g.Prefix
				bestWeight = g.Weight
			}
		}
	}
	return bestPrefix
}
