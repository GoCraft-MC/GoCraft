package customitems

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manager loads all custom item packs and holds the resolved items.
type Manager struct {
	items    []*ResolvedItem
	registry *registry
}

// Load scans packsDir for sub-directories that contain an items.yml, parses
// each one, assigns stable CMD and Bedrock runtime IDs, and returns a Manager.
// If packsDir does not exist, an empty Manager is returned without error.
func Load(packsDir string) (*Manager, error) {
	reg, err := loadRegistry(filepath.Join(packsDir, ".registry.yml"))
	if err != nil {
		return nil, fmt.Errorf("customitems: registry: %w", err)
	}

	entries, err := os.ReadDir(packsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Manager{registry: reg}, nil
		}
		return nil, fmt.Errorf("customitems: reading packs dir: %w", err)
	}

	m := &Manager{registry: reg}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(packsDir, e.Name())
		pack, err := loadPack(dir)
		if err != nil {
			slog.Warn("customitems: skipping pack", "dir", dir, "err", err)
			continue
		}
		slog.Info("customitems: loaded pack",
			"namespace", pack.Def.Namespace, "items", len(pack.Def.Items))

		for id, def := range pack.Def.Items {
			key := pack.Def.Namespace + ":" + id
			ids := reg.assign(key)

			var texData []byte
			if def.Texture != "" {
				texData = pack.Textures[def.Texture]
				if texData == nil {
					slog.Warn("customitems: texture not found",
						"item", key, "texture", def.Texture)
				}
			}

			m.items = append(m.items, &ResolvedItem{
				Namespace:        pack.Def.Namespace,
				ID:               id,
				Def:              def,
				CMD:              ids.CMD,
				BedrockRuntimeID: ids.BedrockRID,
				TextureData:      texData,
			})
		}
	}

	if err := reg.save(); err != nil {
		slog.Warn("customitems: could not save registry", "err", err)
	}

	return m, nil
}

func loadPack(dir string) (*LoadedPack, error) {
	data, err := os.ReadFile(filepath.Join(dir, "items.yml"))
	if err != nil {
		return nil, fmt.Errorf("reading items.yml: %w", err)
	}
	var def PackDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing items.yml: %w", err)
	}
	if def.Namespace == "" {
		return nil, fmt.Errorf("items.yml is missing namespace")
	}

	textures := map[string][]byte{}
	texDir := filepath.Join(dir, "textures")
	texEntries, _ := os.ReadDir(texDir)
	for _, te := range texEntries {
		if te.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(texDir, te.Name()))
		if err == nil {
			textures[te.Name()] = b
		}
	}

	return &LoadedPack{Def: def, Dir: dir, Textures: textures}, nil
}

// Items returns all resolved custom items across all loaded packs.
func (m *Manager) Items() []*ResolvedItem { return m.items }

// IsEmpty returns true if no custom items were loaded.
func (m *Manager) IsEmpty() bool { return len(m.items) == 0 }
