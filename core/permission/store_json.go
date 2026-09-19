package permission

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// JSONStore returns a Store backed by a JSON file at path.
// It is the default backend and requires no external services.
func JSONStore(path string) Store {
	return &jsonStore{path: path}
}

type jsonStore struct{ path string }

func (s *jsonStore) Load() (Document, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		doc := DefaultDocument()
		return doc, s.Save(doc)
	}
	if err != nil {
		return Document{}, err
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return Document{}, errors.New("decode permissions: " + err.Error())
	}
	return doc, nil
}

func (s *jsonStore) Save(doc Document) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *jsonStore) Close() error { return nil }
