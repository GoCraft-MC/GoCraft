package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func loadJavaItemTags(path string, itemIDs map[string]int) (map[string][]string, string) {
	var file itemTagFile
	hash := mustDecode(path, &file)
	idToName := make(map[int]string, len(itemIDs))
	for name, id := range itemIDs {
		idToName[id] = name
	}
	for _, registry := range file.Registries {
		if registry.Name != "minecraft:item" {
			continue
		}
		if registry.RegistrySize != len(itemIDs) {
			panic(fmt.Sprintf("item tag registry size %d, want %d", registry.RegistrySize, len(itemIDs)))
		}
		result := make(map[string][]string, len(registry.Tags))
		for _, tag := range registry.Tags {
			for _, id := range tag.Entries {
				name, ok := idToName[id]
				if !ok {
					panic(fmt.Sprintf("item tag %s has unknown ID %d", tag.Name, id))
				}
				result[tag.Name] = append(result[tag.Name], name)
			}
		}
		return result, hash
	}
	panic("network tags contain no minecraft:item registry")
}

func addPumpkinExtensionTags(path string, tags map[string][]string) string {
	var file struct {
		Item map[string]json.RawMessage `json:"item"`
	}
	hash := mustDecode(path, &file)
	extensions := make(map[string]struct{}, len(compatibilityExtensionIDs))
	for _, itemID := range compatibilityExtensionIDs {
		extensions[itemID] = struct{}{}
	}
	for tag, raw := range file.Item {
		for _, value := range stringValues(raw) {
			itemID := canonicalID(strings.TrimPrefix(value, "#"))
			if _, ok := extensions[itemID]; ok && !strings.HasPrefix(value, "#") {
				tags[tag] = append(tags[tag], itemID)
			}
		}
	}
	return hash
}

func tagsForItem(tags map[string][]string, itemID string) []string {
	var result []string
	for tag, values := range tags {
		for _, value := range values {
			if value == itemID {
				result = append(result, tag)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}
