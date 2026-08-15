package world

import (
	"testing"

	coreworld "GoCraft/core/world"
)

func TestJavaBedsAndOakDoorsMapToVisibleBedrockBlocks(t *testing.T) {
	encoder := NewEncoder()
	tests := []struct {
		block coreworld.Block
		want  string
	}{
		{coreworld.Block{Namespace: `minecraft`, Name: `red_bed`, Properties: map[string]string{`facing`: `north`, `part`: `head`, `occupied`: `false`}}, `minecraft:bed`},
		{coreworld.Block{Namespace: `minecraft`, Name: `oak_door`, Properties: map[string]string{`facing`: `east`, `half`: `lower`, `hinge`: `left`, `open`: `false`, `powered`: `false`}}, `minecraft:wooden_door`},
	}
	for _, test := range tests {
		got, _ := encoder.resolveState(test.block)
		if got != test.want {
			t.Errorf(`%s mapped to %s, want %s`, test.block.ResourceLocation(), got, test.want)
		}
	}
}

func TestTallGrassUsesSingleBlockBedrockFallback(t *testing.T) {
	encoder := NewEncoder()
	lower := coreworld.Block{
		Namespace: "minecraft", Name: "tall_grass",
		Properties: map[string]string{"half": "lower"},
	}
	upper := coreworld.Block{
		Namespace: "minecraft", Name: "tall_grass",
		Properties: map[string]string{"half": "upper"},
	}
	shortGrass := coreworld.Block{Namespace: "minecraft", Name: "short_grass"}

	if got, want := encoder.BlockNetworkID(lower), encoder.BlockNetworkID(shortGrass); got != want {
		t.Fatalf("lower tall grass network hash = %d, want short grass hash %d", got, want)
	}
	if got, want := encoder.BlockNetworkID(upper), encoder.BlockNetworkID(coreworld.Air); got != want {
		t.Fatalf("upper tall grass network hash = %d, want air hash %d", got, want)
	}

	if name, _ := encoder.resolveState(lower); name != "minecraft:short_grass" {
		t.Fatalf("persistent lower state = %q, want minecraft:short_grass", name)
	}
	if name, _ := encoder.resolveState(upper); name != "minecraft:air" {
		t.Fatalf("persistent upper state = %q, want minecraft:air", name)
	}
}
