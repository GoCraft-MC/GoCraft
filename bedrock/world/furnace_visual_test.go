package world

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestFurnaceVisualVariantsUseBedrockLitIdentifiers(t *testing.T) {
	encoder := NewEncoder()
	tests := []struct {
		name      string
		blockName string
		lit       string
		wantName  string
	}{
		{name: "unlit furnace", blockName: "furnace", lit: "false", wantName: "minecraft:furnace"},
		{name: "lit furnace", blockName: "furnace", lit: "true", wantName: "minecraft:lit_furnace"},
		{name: "unlit blast furnace", blockName: "blast_furnace", lit: "false", wantName: "minecraft:blast_furnace"},
		{name: "lit blast furnace", blockName: "blast_furnace", lit: "true", wantName: "minecraft:lit_blast_furnace"},
		{name: "unlit smoker", blockName: "smoker", lit: "false", wantName: "minecraft:smoker"},
		{name: "lit smoker", blockName: "smoker", lit: "true", wantName: "minecraft:lit_smoker"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block := coreworld.Block{
				Namespace: "minecraft",
				Name:      test.blockName,
				Properties: map[string]string{
					"facing": "north",
					"lit":    test.lit,
				},
			}
			name, _ := encoder.resolveState(block)
			if name != test.wantName {
				t.Fatalf("name = %q, want %q", name, test.wantName)
			}
		})
	}
}
