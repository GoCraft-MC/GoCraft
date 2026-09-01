package main

import (
	"fmt"
	"sort"
	"strings"
)

func generate(paths options) catalogue {
	var hashes inputHashes
	var report map[string]itemReportEntry
	hashes.report = mustDecode(paths.itemsReport, &report)
	var ids struct {
		Version string         `json:"_gocraft_version"`
		Entries map[string]int `json:"entries"`
	}
	hashes.itemIDs = mustDecode(paths.itemIDs, &ids)
	if ids.Version != minecraftVersion || len(report) != len(ids.Entries) {
		panic(fmt.Sprintf("item source mismatch: version=%q report=%d IDs=%d", ids.Version, len(report), len(ids.Entries)))
	}
	for itemID := range ids.Entries {
		if _, ok := report[itemID]; !ok {
			panic(fmt.Sprintf("Mojang report is missing %s", itemID))
		}
	}
	tags, tagHash := loadJavaItemTags(paths.itemTags, ids.Entries)
	hashes.itemTags = tagHash
	var pumpkin map[string]pumpkinItemEntry
	hashes.pumpkinItems = mustDecode(paths.pumpkinItems, &pumpkin)
	hashes.pumpkinTags = addPumpkinExtensionTags(paths.pumpkinTags, tags)

	items := make(map[string]definition, len(report)+len(compatibilityExtensionIDs))
	for itemID, entry := range report {
		items[itemID] = buildDefinition(itemID, entry.Components, tagsForItem(tags, itemID))
	}
	for _, itemID := range compatibilityExtensionIDs {
		entry, ok := pumpkin[strings.TrimPrefix(itemID, "minecraft:")]
		if !ok {
			panic(fmt.Sprintf("Pumpkin source is missing %s", itemID))
		}
		items[itemID] = buildDefinition(itemID, entry.Components, tagsForItem(tags, itemID))
	}
	var fuelFile struct {
		Version string         `json:"_gocraft_version"`
		Fuels   map[string]int `json:"fuels"`
	}
	hashes.fuels = mustDecode(paths.fuels, &fuelFile)
	if fuelFile.Version != minecraftVersion {
		panic(fmt.Sprintf("fuel version %q, want %s", fuelFile.Version, minecraftVersion))
	}
	for itemID, ticks := range fuelFile.Fuels {
		value, ok := items[itemID]
		if !ok {
			panic(fmt.Sprintf("fuel references unknown item %s", itemID))
		}
		value.FuelTicks = ticks
		items[itemID] = value
	}
	addGeneratedItemCategories(items, tags)
	for tag, values := range tags {
		sort.Strings(values)
		tags[tag] = compactStrings(values)
	}
	return newCatalogue(items, tags, hashes)
}
