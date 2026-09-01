package main

import "encoding/json"

type itemReportEntry struct {
	Components map[string]json.RawMessage `json:"components"`
}

type pumpkinItemEntry struct {
	Components map[string]json.RawMessage `json:"components"`
}

type attributeModifier struct {
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Operation string  `json:"operation"`
	Slot      string  `json:"slot"`
}

type itemTagFile struct {
	Registries []struct {
		Name         string `json:"name"`
		RegistrySize int    `json:"registry_size"`
		Tags         []struct {
			Name    string `json:"name"`
			Entries []int  `json:"entries"`
		} `json:"tags"`
	} `json:"registries"`
}

type options struct {
	itemsReport  string
	itemIDs      string
	itemTags     string
	fuels        string
	pumpkinItems string
	pumpkinTags  string
	output       string
}

type inputHashes struct {
	report, itemIDs, itemTags, fuels, pumpkinItems, pumpkinTags string
}
