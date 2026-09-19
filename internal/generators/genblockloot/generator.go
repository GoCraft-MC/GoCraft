// Command genblockloot builds GoCraft's compact, versioned block-loot bundle
// from an official Minecraft server/client jar and Pumpkin's generated block
// metadata. The jar supplies exact versioned loot tables and tags. Pumpkin's
// state flag supplies requires-correct-tool-for-drops, which vanilla's data
// pack does not expose as JSON.
package genblockloot

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

const toolRequiredFlag uint16 = 1 << 2

type tagFile struct {
	Replace bool              `json:"replace"`
	Values  []json.RawMessage `json:"values"`
}

type bundle struct {
	Version      string                     `json:"version"`
	LootTables   map[string]json.RawMessage `json:"loot_tables"`
	BlockTags    map[string][]string        `json:"block_tags"`
	ItemTags     map[string][]string        `json:"item_tags"`
	ToolRequired []string                   `json:"tool_required"`
}

type pumpkinBlocks struct {
	Blocks []struct {
		Name   string `json:"name"`
		States []struct {
			StateFlags uint16 `json:"state_flags"`
		} `json:"states"`
	} `json:"blocks"`
}

func Generate() error {
	jarPath := viper.GetString("jar")
	pumpkinPath := viper.GetString("pumpkin-blocks")
	outputPath := viper.GetString("output")
	version := viper.GetString("version")

	archive, err := zip.OpenReader(jarPath)
	if err != nil {
		return fmt.Errorf("failed to open Minecraft jar: %v", err)
	}

	defer archive.Close()

	lootTables := make(map[string]json.RawMessage)
	rawBlockTags := make(map[string]tagFile)
	rawItemTags := make(map[string]tagFile)
	for _, entry := range archive.File {
		switch {
		case strings.HasPrefix(entry.Name, "data/minecraft/loot_table/blocks/") && strings.HasSuffix(entry.Name, ".json"):
			name := strings.TrimSuffix(path.Base(entry.Name), ".json")
			e, err := readJSONEntry(entry)
			if err != nil {
				return err
			}

			lootTables["minecraft:"+name] = e

		case strings.HasPrefix(entry.Name, "data/minecraft/tags/block/") && strings.HasSuffix(entry.Name, ".json"):
			key := tagKey(entry.Name, "data/minecraft/tags/block/")
			e, err := readTagEntry(entry)
			if err != nil {
				return err
			}

			rawBlockTags[key] = *e

		case strings.HasPrefix(entry.Name, "data/minecraft/tags/item/") && strings.HasSuffix(entry.Name, ".json"):
			key := tagKey(entry.Name, "data/minecraft/tags/item/")
			e, err := readTagEntry(entry)
			if err != nil {
				return err
			}

			rawItemTags[key] = *e
		}
	}

	if len(lootTables) == 0 {
		return fmt.Errorf("no block loot tables found in: %s", jarPath)
	}

	pumpkinData, err := os.ReadFile(pumpkinPath)
	if err != nil {
		return fmt.Errorf("failed to read Pumpkin blocks: %v", err)
	}

	var pumpkin pumpkinBlocks
	if err := json.Unmarshal(pumpkinData, &pumpkin); err != nil {
		return fmt.Errorf("failed to decode Pumpkin blocks: %v", err)
	}

	toolRequired := make([]string, 0)
	for _, block := range pumpkin.Blocks {
		blockID := namespaced(block.Name)
		// Pumpkin may be newer than the target jar. Keep its harvest flag only
		// for blocks present in the exact target-version loot registry so the
		// bundle can never expose post-1.21.4 content.
		if _, presentInTargetVersion := lootTables[blockID]; !presentInTargetVersion {
			continue
		}

		for _, state := range block.States {
			if state.StateFlags&toolRequiredFlag != 0 {
				toolRequired = append(toolRequired, blockID)
				break
			}
		}
	}
	sort.Strings(toolRequired)

	result := bundle{
		Version:      version,
		LootTables:   lootTables,
		BlockTags:    resolveTags(rawBlockTags),
		ItemTags:     resolveTags(rawItemTags),
		ToolRequired: toolRequired,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode bundle: %v", err)
	}

	encoded = append(encoded, '\n')
	if err := os.WriteFile(outputPath, encoded, 0o644); err != nil {
		return fmt.Errorf("failed to write output %s: %v", outputPath, err)
	}

	fmt.Printf("generated %s: %d loot tables, %d block tags, %d item tags, %d tool-required blocks\n",
		outputPath, len(lootTables), len(result.BlockTags), len(result.ItemTags), len(toolRequired))

	return nil
}

func readJSONEntry(entry *zip.File) (json.RawMessage, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", entry.Name, err)
	}

	defer reader.Close()
	var compact bytes.Buffer
	bytes, err := mustReadAll(reader, entry.Name)
	if err != nil {
		return nil, err
	}

	if err := json.Compact(&compact, bytes); err != nil {
		return nil, fmt.Errorf("failed to compact %s: %v", entry.Name, err)
	}

	return compact.Bytes(), nil
}

func readTagEntry(entry *zip.File) (*tagFile, error) {
	raw, err := readJSONEntry(entry)
	if err != nil {
		return nil, err
	}

	var tag tagFile
	if err := json.Unmarshal(raw, &tag); err != nil {
		return nil, fmt.Errorf("failed to decode tag %s: %v", entry.Name, err)
	}

	return &tag, nil
}

func mustReadAll(reader interface{ Read([]byte) (int, error) }, name string) ([]byte, error) {
	var output bytes.Buffer
	if _, err := output.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", name, err)
	}

	return output.Bytes(), nil
}

func tagKey(entryName, prefix string) string {
	relative := strings.TrimSuffix(strings.TrimPrefix(entryName, prefix), ".json")
	return namespaced(relative)
}

func resolveTags(raw map[string]tagFile) map[string][]string {
	resolved := make(map[string][]string, len(raw))
	visiting := make(map[string]bool)

	var resolve func(string) []string
	resolve = func(key string) []string {
		key = namespaced(strings.TrimPrefix(key, "#"))
		if values, ok := resolved[key]; ok {
			return values
		}

		if visiting[key] {
			fatalf("cyclic tag reference at %s", key)
		}

		visiting[key] = true
		set := make(map[string]struct{})
		for _, encoded := range raw[key].Values {
			var value string
			if err := json.Unmarshal(encoded, &value); err != nil {
				var optional struct {
					ID string `json:"id"`
				}

				if err := json.Unmarshal(encoded, &optional); err != nil || optional.ID == "" {
					fatalf("invalid value in tag %s: %s", key, encoded)
				}

				value = optional.ID
			}

			if strings.HasPrefix(value, "#") {
				for _, nested := range resolve(value) {
					set[nested] = struct{}{}
				}
				continue
			}

			set[namespaced(value)] = struct{}{}
		}

		delete(visiting, key)
		values := make([]string, 0, len(set))
		for value := range set {
			values = append(values, value)
		}

		sort.Strings(values)
		resolved[key] = values
		return values
	}

	for key := range raw {
		resolve(key)
	}

	return resolved
}

func namespaced(value string) string {
	if strings.Contains(value, ":") {
		return value
	}
	return "minecraft:" + value
}

// TODO(@s0cks): remove
func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "genblockloot: "+format+"\n", args...)
	os.Exit(1)
}
