package main

import (
	"strings"
	"testing"
)

func TestCategorySectionDoesNotMatchCategoryNameSuffix(t *testing.T) {
	body := `
            underground_water_creature: &[Spawner {
                r#type: "minecraft:glow_squid",
                min_count: 4i32,
                max_count: 6i32,
            }],
            water_ambient: &[Spawner {
                r#type: "minecraft:cod",
                min_count: 3i32,
                max_count: 6i32,
            }],
            water_creature: &[Spawner {
                r#type: "minecraft:squid",
                min_count: 1i32,
                max_count: 4i32,
            }],`
	section, ok := categorySection(body, "water_creature", "")
	if !ok {
		t.Fatal("water_creature section not found")
	}
	if !strings.Contains(section, "minecraft:squid") || strings.Contains(section, "minecraft:cod") || strings.Contains(section, "minecraft:glow_squid") {
		t.Fatalf("water_creature section crossed category boundaries:\n%s", section)
	}
}
