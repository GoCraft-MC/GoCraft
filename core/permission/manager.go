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
