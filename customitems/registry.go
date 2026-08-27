package customitems

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	cmdStart           = 30100 // start of the CustomModelData range (matches CustomiZer)
	bedrockRIDStart    = 5000  // well above the highest vanilla Bedrock runtime ID (~750)
)

// registry persists "namespace:id" → {cmd, bedrockRID} assignments across
// restarts so IDs are stable (changing them would break saved items in worlds).
type registry struct {
	path    string
	entries map[string]registryEntry // "namespace:id" → assigned IDs

	nextCMD    int
	nextBedRID int16
}

type registryEntry struct {
	CMD          int   `yaml:"cmd"`
	BedrockRID   int16 `yaml:"bedrock_rid"`
}

// registryFile is the on-disk YAML structure.
type registryFile struct {
	Entries map[string]registryEntry `yaml:"entries"`
}

func loadRegistry(path string) (*registry, error) {
	r := &registry{
		path:       path,
		entries:    map[string]registryEntry{},
		nextCMD:    cmdStart,
		nextBedRID: bedrockRIDStart,
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}

	var f registryFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Entries != nil {
		r.entries = f.Entries
	}

	// Find the next free IDs from what's already assigned.
	for _, e := range r.entries {
		if e.CMD >= r.nextCMD {
			r.nextCMD = e.CMD + 1
		}
		if e.BedrockRID >= r.nextBedRID {
			r.nextBedRID = e.BedrockRID + 1
		}
	}
	return r, nil
}

// assign returns the IDs for a key, creating new ones if they do not exist.
func (r *registry) assign(key string) registryEntry {
	if e, ok := r.entries[key]; ok {
		return e
	}
	e := registryEntry{
		CMD:        r.nextCMD,
		BedrockRID: r.nextBedRID,
	}
	r.nextCMD++
	r.nextBedRID++
	r.entries[key] = e
	return e
}

func (r *registry) save() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("registry mkdir: %w", err)
	}
	data, err := yaml.Marshal(registryFile{Entries: r.entries})
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o644)
}
