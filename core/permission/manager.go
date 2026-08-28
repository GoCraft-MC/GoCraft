package permission

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Manager struct {
	mu       sync.RWMutex
	path     string
	document Document
}

func Load(path string) (*Manager, error) {
	manager := &Manager{path: path, document: DefaultDocument()}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return manager, manager.persist(manager.document)
	}
	if err != nil {
		return nil, err
	}
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, errors.New("decode permissions: " + err.Error())
	}
	if err := validateDocument(document); err != nil {
		return nil, err
	}
	manager.document = document
	return manager, nil
}

func NewMemory() *Manager {
	return &Manager{document: DefaultDocument()}
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
	if err := m.persist(document); err != nil {
		return err
	}
	m.document = document
	return nil
}

// Reload replaces the active document from its configured file atomically.
func (m *Manager) Reload() error {
	if m.path == "" {
		return nil
	}
	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		return errors.New("decode permissions: " + err.Error())
	}
	if err := validateDocument(document); err != nil {
		return err
	}
	m.mu.Lock()
	m.document = cloneDocument(document)
	m.mu.Unlock()
	return nil
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

func (m *Manager) persist(document Document) error {
	if m.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	temporary := m.path + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, m.path)
}
