package customitems

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// BuildBedrockPack generates a Bedrock Edition .mcaddon in memory.
//
// The archive contains two top-level packs:
//   - gocraft_rp/ — resource pack (textures + item_texture.json)
//   - gocraft_bp/ — behavior pack (one items/<ns>/<id>.json per custom item)
//
// Returns nil when the manager has no items.
func (m *Manager) BuildBedrockPack() ([]byte, error) {
	if m.IsEmpty() {
		return nil, nil
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	const (
		rpUUID    = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
		rpModUUID = "c3d4e5f6-a7b8-9012-cdef-123456789012"
		bpUUID    = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
		bpModUUID = "d4e5f6a7-b8c9-0123-defa-234567890123"
	)

	if err := writeBedrockRP(zw, rpUUID, rpModUUID, m.items); err != nil {
		return nil, fmt.Errorf("bedrock RP: %w", err)
	}
	if err := writeBedrockBP(zw, bpUUID, bpModUUID, rpUUID, m.items); err != nil {
		return nil, fmt.Errorf("bedrock BP: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeBedrockRP(zw *zip.Writer, uuid, modUUID string, items []*ResolvedItem) error {
	p := "gocraft_rp/"

	manifest := map[string]any{
		"format_version": 2,
		"header": map[string]any{
			"name": "GoCraft Custom Items RP", "description": "auto-generated",
			"uuid": uuid, "version": []int{1, 0, 0},
			"min_engine_version": []int{1, 16, 0},
		},
		"modules": []map[string]any{
			{"type": "resources", "uuid": modUUID, "version": []int{1, 0, 0}},
		},
	}
	if err := writeZipJSON(zw, p+"manifest.json", manifest); err != nil {
		return err
	}

	// item_texture.json — maps texture keys to PNG paths.
	textureData := map[string]any{}
	for _, item := range items {
		key := item.Namespace + "_" + item.ID
		textureData[key] = map[string]any{
			"textures": "textures/items/" + item.Namespace + "/" + item.ID,
		}
	}
	if err := writeZipJSON(zw, p+"textures/item_texture.json", map[string]any{
		"resource_pack_name": "gocraft",
		"texture_name":       "atlas.items",
		"texture_data":       textureData,
	}); err != nil {
		return err
	}

	// Texture PNGs.
	for _, item := range items {
		if len(item.TextureData) == 0 {
			continue
		}
		path := p + "textures/items/" + item.Namespace + "/" + item.ID + ".png"
		if err := writeZipEntry(zw, path, item.TextureData); err != nil {
			return err
		}
	}
	return nil
}

func writeBedrockBP(zw *zip.Writer, uuid, modUUID, rpUUID string, items []*ResolvedItem) error {
	p := "gocraft_bp/"

	manifest := map[string]any{
		"format_version": 2,
		"header": map[string]any{
			"name": "GoCraft Custom Items BP", "description": "auto-generated",
			"uuid": uuid, "version": []int{1, 0, 0},
			"min_engine_version": []int{1, 16, 220},
		},
		"modules": []map[string]any{
			{"type": "data", "uuid": modUUID, "version": []int{1, 0, 0}},
		},
		"dependencies": []map[string]any{
			{"uuid": rpUUID, "version": []int{1, 0, 0}},
		},
	}
	if err := writeZipJSON(zw, p+"manifest.json", manifest); err != nil {
		return err
	}

	for _, item := range items {
		maxStack := item.Def.MaxStackSize
		if maxStack <= 0 {
			maxStack = 64
		}
		def := map[string]any{
			"format_version": "1.16.220",
			"minecraft:item": map[string]any{
				"description": map[string]any{
					"identifier": item.Key(), "category": "items",
				},
				"components": map[string]any{
					"minecraft:display_name":   map[string]any{"value": stripTags(item.Def.DisplayName)},
					"minecraft:icon":           map[string]any{"texture": item.Namespace + "_" + item.ID},
					"minecraft:max_stack_size": maxStack,
					"minecraft:hand_equipped":  item.Def.HandEquipped,
				},
			},
		}
		path := fmt.Sprintf("%sitems/%s/%s.json", p, item.Namespace, item.ID)
		if err := writeZipJSON(zw, path, def); err != nil {
			return err
		}
	}
	return nil
}

func writeZipJSON(zw *zip.Writer, name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeZipEntry(zw, name, data)
}

// stripTags removes MiniMessage / §-color tags to produce plain text for Bedrock.
func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '<':
			inTag = true
		case s[i] == '>':
			inTag = false
		case s[i] == '\u00a7' && i+1 < len(s): // § section sign
			i++ // skip the code char
		case !inTag:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
