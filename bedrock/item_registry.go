package bedrock

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// pumpkinItemComponentsGZIP comes from Pumpkin's Bedrock 1.26.40 assets. The
// client needs these definitions together with runtime_item_states.json: an
// empty ItemRegistry leaves Creative search indexing and data-driven item use
// (including food) with unresolved runtime IDs/components.
//
//go:embed item_components.nbt.gz
var pumpkinItemComponentsGZIP []byte

var pumpkinBedrockItemRegistry = mustBuildPumpkinBedrockItemRegistry()

func bedrockItemRegistry() []protocol.ItemEntry {
	return pumpkinBedrockItemRegistry
}

func mustBuildPumpkinBedrockItemRegistry() []protocol.ItemEntry {
	compressed, err := gzip.NewReader(bytes.NewReader(pumpkinItemComponentsGZIP))
	if err != nil {
		panic(fmt.Errorf("bedrock: open embedded Pumpkin item components: %w", err))
	}
	defer compressed.Close()

	components := make(map[string]any)
	if err := nbt.NewDecoderWithEncoding(compressed, nbt.BigEndian).Decode(&components); err != nil {
		panic(fmt.Errorf("bedrock: decode embedded Pumpkin item components: %w", err))
	}

	entries := make([]protocol.ItemEntry, 0, len(pumpkinItemRegistryDataEntries))
	for _, item := range pumpkinItemRegistryDataEntries {
		data := map[string]any{}
		if componentData, found := components[item.name]; found {
			compound, ok := componentData.(map[string]any)
			if !ok {
				panic(fmt.Sprintf("bedrock: item components for %s are %T, want compound", item.name, componentData))
			}
			data = compound
		} else if item.componentBased {
			panic(fmt.Sprintf("bedrock: component-based item %s has no component definition", item.name))
		}
		entries = append(entries, protocol.ItemEntry{
			Name:           item.name,
			RuntimeID:      item.runtimeID,
			ComponentBased: item.componentBased,
			Version:        item.version,
			Data:           data,
		})
	}
	return entries
}
