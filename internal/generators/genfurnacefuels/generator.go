// Command genfurnacefuels converts Pumpkin's generated fuel table into a
// resource-location keyed catalogue filtered to GoCraft's Java 1.21.4 items.
package genfurnacefuels

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"

	"github.com/spf13/viper"
)

type pumpkinItem struct {
	ID int `json:"id"`
}

type javaItems struct {
	Version string         `json:"_gocraft_version"`
	Entries map[string]int `json:"entries"`
}

type fuelCatalogue struct {
	Version string         `json:"_gocraft_version"`
	Source  string         `json:"_source"`
	Fuels   map[string]int `json:"fuels"`
}

var fuelEntry = regexp.MustCompile(`\((\d+)u16,\s*(\d+)u16\)`)

func Generate() error {
	pumpkinItemsPath := viper.GetString("pumpkin-items")
	pumpkinFuelsPath := viper.GetString("pumpkin-fuels")
	javaItemsPath := viper.GetString("java-items")
	outPath := viper.GetString("out")

	if pumpkinItemsPath == "" || pumpkinFuelsPath == "" {
		os.Exit(2)
	}

	var pumpkinItems map[string]pumpkinItem
	mustDecode(pumpkinItemsPath, &pumpkinItems)
	var java javaItems
	mustDecode(javaItemsPath, &java)
	if java.Version != "1.21.4" {
		panic(fmt.Sprintf("Java item catalogue version %q, want 1.21.4", java.Version))
	}

	idToName := make(map[int]string, len(pumpkinItems))
	for name, item := range pumpkinItems {
		idToName[item.ID] = "minecraft:" + name
	}
	rust, err := os.ReadFile(pumpkinFuelsPath)
	if err != nil {
		panic(err)
	}
	fuels := make(map[string]int)
	for _, match := range fuelEntry.FindAllSubmatch(rust, -1) {
		id, _ := strconv.Atoi(string(match[1]))
		ticks, _ := strconv.Atoi(string(match[2]))
		name := idToName[id]
		if name == "" {
			panic(fmt.Sprintf("Pumpkin fuel item ID %d has no item name", id))
		}
		if _, validForJava := java.Entries[name]; validForJava {
			fuels[name] = ticks
		}
	}
	if len(fuels) < 200 {
		panic(fmt.Sprintf("only %d Java 1.21.4 fuels resolved", len(fuels)))
	}

	// encoding/json sorts map keys, producing a deterministic checked-in asset.
	payload, err := json.MarshalIndent(fuelCatalogue{
		Version: "1.21.4",
		Source:  "Pumpkin generated fuel durations, filtered to Java 1.21.4 items",
		Fuels:   fuels,
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(outPath, payload, 0o644); err != nil {
		panic(err)
	}

	names := make([]string, 0, len(fuels))
	for name := range fuels {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Printf("wrote %d Java 1.21.4 fuels to %s\n", len(names), outPath)
	return nil
}

func mustDecode(path string, target any) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		panic(fmt.Errorf("decode %s: %w", path, err))
	}
}
