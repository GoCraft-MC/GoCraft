package customitems

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
)

// BuildJavaPack generates a Java Edition resource pack ZIP in memory.
// It returns the raw ZIP bytes and the SHA-1 hash of those bytes (used as the
// pack hash in the resource_pack_push packet so clients can cache the pack).
// Returns nil, zero, nil when the manager has no items loaded.
func (m *Manager) BuildJavaPack() ([]byte, [20]byte, error) {
	if m.IsEmpty() {
		return nil, [20]byte{}, nil
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// pack.mcmeta — required by the client to recognise the archive as a pack.
	// pack_format 34 = Minecraft 1.21.4.
	if err := writeZipEntry(zw, "pack.mcmeta", []byte(
		`{"pack":{"pack_format":34,"description":"GoCraft Custom Items"}}`,
	)); err != nil {
		return nil, [20]byte{}, err
	}

	// Group items by base material so we emit one override file per material.
	type override struct {
		Predicate map[string]int `json:"predicate"`
		Model     string         `json:"model"`
	}
	byMaterial := map[string][]override{}
	for _, item := range m.items {
		mat := strings.ToLower(item.Def.Material)
		byMaterial[mat] = append(byMaterial[mat], override{
			Predicate: map[string]int{"custom_model_data": item.CMD},
			Model:     item.Namespace + ":item/" + item.ID,
		})
	}

	// Write the base material model files with CustomModelData overrides.
	for mat, overrides := range byMaterial {
		type modelJSON struct {
			Parent    string            `json:"parent"`
			Textures  map[string]string `json:"textures,omitempty"`
			Overrides []override        `json:"overrides"`
		}
		model := modelJSON{
			Parent:    javaVanillaParent(mat),
			Textures:  map[string]string{"layer0": "minecraft:item/" + mat},
			Overrides: overrides,
		}
		data, err := json.MarshalIndent(model, "", "  ")
		if err != nil {
			return nil, [20]byte{}, err
		}
		path := fmt.Sprintf("assets/minecraft/models/item/%s.json", mat)
		if err := writeZipEntry(zw, path, data); err != nil {
			return nil, [20]byte{}, err
		}
	}

	// Write per-item model JSON and texture PNG.
	for _, item := range m.items {
		parent := item.Def.Parent
		if parent == "" {
			parent = javaVanillaParent(strings.ToLower(item.Def.Material))
		}

		type itemModel struct {
			Parent   string            `json:"parent"`
			Textures map[string]string `json:"textures"`
		}
		model := itemModel{
			Parent:   parent,
			Textures: map[string]string{"layer0": item.Namespace + ":item/" + item.ID},
		}
		data, err := json.MarshalIndent(model, "", "  ")
		if err != nil {
			return nil, [20]byte{}, err
		}
		modelPath := fmt.Sprintf("assets/%s/models/item/%s.json", item.Namespace, item.ID)
		if err := writeZipEntry(zw, modelPath, data); err != nil {
			return nil, [20]byte{}, err
		}

		if len(item.TextureData) > 0 {
			texPath := fmt.Sprintf("assets/%s/textures/item/%s.png", item.Namespace, item.ID)
			if err := writeZipEntry(zw, texPath, item.TextureData); err != nil {
				return nil, [20]byte{}, err
			}
		}
	}

	if err := zw.Close(); err != nil {
		return nil, [20]byte{}, err
	}

	raw := buf.Bytes()
	return raw, sha1.Sum(raw), nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// javaVanillaParent returns the standard Java model parent for the given
// lowercase material name. Swords and tools use "item/handheld" so the item
// is held correctly; everything else uses "item/generated" (flat sprite).
func javaVanillaParent(mat string) string {
	switch mat {
	case "wooden_sword", "stone_sword", "iron_sword", "golden_sword",
		"diamond_sword", "netherite_sword",
		"wooden_pickaxe", "stone_pickaxe", "iron_pickaxe", "golden_pickaxe",
		"diamond_pickaxe", "netherite_pickaxe",
		"wooden_axe", "stone_axe", "iron_axe", "golden_axe",
		"diamond_axe", "netherite_axe",
		"wooden_shovel", "stone_shovel", "iron_shovel", "golden_shovel",
		"diamond_shovel", "netherite_shovel",
		"wooden_hoe", "stone_hoe", "iron_hoe", "golden_hoe",
		"diamond_hoe", "netherite_hoe":
		return "item/handheld"
	}
	return "item/generated"
}
