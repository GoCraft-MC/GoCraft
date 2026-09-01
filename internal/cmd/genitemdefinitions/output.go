package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

func newCatalogue(items map[string]definition, tags map[string][]string, hashes inputHashes) catalogue {
	extensions := append([]string(nil), compatibilityExtensionIDs...)
	sort.Strings(extensions)
	return catalogue{
		Schema: 1, MinecraftVersion: minecraftVersion,
		Comment: "Canonical static item metadata. Generated; do not edit by hand.",
		Sources: []sourceDescription{
			{Name: "Mojang server data generator Item List", Version: minecraftVersion, ArtifactSHA1: minecraftServerSHA1, InputSHA256: hashes.report, Description: "Official reports/items.json default components"},
			{Name: "GoCraft Java item IDs", Version: minecraftVersion, InputSHA256: hashes.itemIDs, Description: "Exact base item coverage"},
			{Name: "GoCraft flattened Java item tags", Version: minecraftVersion, InputSHA256: hashes.itemTags, Description: "Categories and repair ingredients"},
			{Name: "GoCraft generated furnace fuels", Version: minecraftVersion, InputSHA256: hashes.fuels, Description: "Fuel burn duration in ticks"},
			{Name: "Pumpkin generated item components", InputSHA256: hashes.pumpkinItems, Description: "Listed compatibility armor only"},
			{Name: "Pumpkin generated item tags", InputSHA256: hashes.pumpkinTags, Description: "Compatibility armor tags only"},
		},
		CompatibilityExtensions: extensions, Items: items, Tags: tags,
	}
}

func writeCatalogue(path string, output catalogue) {
	// The generator and recorded hashes are the reviewable source. Compact JSON
	// keeps the generated-data commit separate from handwritten line budgets.
	payload, err := json.Marshal(output)
	if err != nil {
		panic(err)
	}
	payload = append(payload, '\n')
	if err := os.MkdirAll(directory(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d item definitions and %d tags to %s\n", len(output.Items), len(output.Tags), path)
}
