// Command genblockloot builds GoCraft's compact, versioned block-loot bundle
// from an official Minecraft server/client jar and Pumpkin's generated block
// metadata. The jar supplies exact versioned loot tables and tags. Pumpkin's
// state flag supplies requires-correct-tool-for-drops, which vanilla's data
// pack does not expose as JSON.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
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

func main() {
	jarPath := flag.String("jar", "", "official Minecraft jar containing data/minecraft")
	pumpkinPath := flag.String("pumpkin-blocks", "", "Pumpkin assets/blocks.json")
	outputPath := flag.String("output", "internal/gamedata/java/1.21.4/block_loot.json", "output bundle")
	version := flag.String("version", "1.21.4", "Minecraft Java version")
	flag.Parse()
	if *jarPath == "" || *pumpkinPath == "" {
		fatalf("-jar and -pumpkin-blocks are required")
	}

	archive, err := zip.OpenReader(*jarPath)
	if err != nil {
		fatalf("open Minecraft jar: %v", err)
	}
	defer archive.Close()

	lootTables := make(map[string]json.RawMessage)
	rawBlockTags := make(map[string]tagFile)
	rawItemTags := make(map[string]tagFile)
	for _, entry := range archive.File {
		switch {
		case strings.HasPrefix(entry.Name, "data/minecraft/loot_table/blocks/") && strings.HasSuffix(entry.Name, ".json"):
			name := strings.TrimSuffix(path.Base(entry.Name), ".json")
			lootTables["minecraft:"+name] = readJSONEntry(entry)
		case strings.HasPrefix(entry.Name, "data/minecraft/tags/block/") && strings.HasSuffix(entry.Name, ".json"):
			key := tagKey(entry.Name, "data/minecraft/tags/block/")
			rawBlockTags[key] = readTagEntry(entry)
		case strings.HasPrefix(entry.Name, "data/minecraft/tags/item/") && strings.HasSuffix(entry.Name, ".json"):
			key := tagKey(entry.Name, "data/minecraft/tags/item/")
			rawItemTags[key] = readTagEntry(entry)
		}
	}
	if len(lootTables) == 0 {
		fatalf("no block loot tables found in %s", *jarPath)
	}

	pumpkinData, err := os.ReadFile(*pumpkinPath)
	if err != nil {
		fatalf("read Pumpkin blocks: %v", err)
	}
	var pumpkin pumpkinBlocks
	if err := json.Unmarshal(pumpkinData, &pumpkin); err != nil {
		fatalf("decode Pumpkin blocks: %v", err)
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
		Version:      *version,
		LootTables:   lootTables,
		BlockTags:    resolveTags(rawBlockTags),
		ItemTags:     resolveTags(rawItemTags),
		ToolRequired: toolRequired,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatalf("encode bundle: %v", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(*outputPath, encoded, 0o644); err != nil {
		fatalf("write %s: %v", *outputPath, err)
	}
	fmt.Printf("generated %s: %d loot tables, %d block tags, %d item tags, %d tool-required blocks\n",
		*outputPath, len(lootTables), len(result.BlockTags), len(result.ItemTags), len(toolRequired))
}

func readJSONEntry(entry *zip.File) json.RawMessage {
	reader, err := entry.Open()
	if err != nil {
		fatalf("open %s: %v", entry.Name, err)
	}
	defer reader.Close()
	var compact bytes.Buffer
	if err := json.Compact(&compact, mustReadAll(reader, entry.Name)); err != nil {
		fatalf("compact %s: %v", entry.Name, err)
	}
	return compact.Bytes()
}

func readTagEntry(entry *zip.File) tagFile {
	raw := readJSONEntry(entry)
	var tag tagFile
	if err := json.Unmarshal(raw, &tag); err != nil {
		fatalf("decode tag %s: %v", entry.Name, err)
	}
	return tag
}

func mustReadAll(reader interface{ Read([]byte) (int, error) }, name string) []byte {
	var output bytes.Buffer
	if _, err := output.ReadFrom(reader); err != nil {
		fatalf("read %s: %v", name, err)
	}
	return output.Bytes()
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

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "genblockloot: "+format+"\n", args...)
	os.Exit(1)
}
