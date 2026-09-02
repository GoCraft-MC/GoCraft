package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

func componentInt(components map[string]json.RawMessage, name string, fallback int) int {
	raw := components[name]
	if !componentPresent(raw) {
		return fallback
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		panic(fmt.Errorf("decode %s: %w", name, err))
	}
	return value
}

func componentString(components map[string]json.RawMessage, name, fallback string) string {
	raw := components[name]
	if !componentPresent(raw) {
		return fallback
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		panic(fmt.Errorf("decode %s: %w", name, err))
	}
	return value
}

func componentPresent(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func mustUnmarshalComponent(itemID, component string, raw json.RawMessage, target any) {
	if err := json.Unmarshal(raw, target); err != nil {
		panic(fmt.Errorf("decode %s %s component: %w", itemID, component, err))
	}
}

func stringValues(raw json.RawMessage) []string {
	if !componentPresent(raw) {
		return nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		panic(fmt.Errorf("decode string or string list: %w", err))
	}
	return values
}

func canonicalIngredient(value string) string {
	if strings.HasPrefix(value, "#") {
		return "#" + canonicalID(strings.TrimPrefix(value, "#"))
	}
	return canonicalID(value)
}

func canonicalID(value string) string {
	if strings.Contains(value, ":") {
		return value
	}
	return "minecraft:" + value
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func cleanFloat(value float64) float32 {
	return float32(math.Round(value*10_000) / 10_000)
}
