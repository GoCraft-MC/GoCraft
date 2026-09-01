package main

import "strings"

// addGeneratedItemCategories records vanilla classifications that Mojang does
// not expose as item tags. They are resolved during generation, never by hot
// gameplay paths, and live in the same versioned canonical tag index.
func addGeneratedItemCategories(items map[string]definition, tags map[string][]string) {
	suffixes := map[string]string{
		"gocraft:dyes":            "_dye",
		"gocraft:banner_patterns": "_banner_pattern",
	}
	for itemID := range items {
		for tag, suffix := range suffixes {
			if strings.HasSuffix(itemID, suffix) {
				tags[tag] = append(tags[tag], itemID)
			}
		}
	}
}
