package world

import "testing"

func TestWaxCopper(t *testing.T) {
	block := Block{Namespace: "minecraft", Name: "weathered_copper_bulb", Properties: map[string]string{"lit": "true"}}
	waxed, ok := WaxCopper(block)
	if !ok || waxed.ResourceLocation() != "minecraft:waxed_weathered_copper_bulb" || waxed.Properties["lit"] != "true" {
		t.Fatalf("waxed block = %+v, ok=%v", waxed, ok)
	}
	waxed.Properties["lit"] = "false"
	if block.Properties["lit"] != "true" {
		t.Fatal("waxing aliased the source properties")
	}
}

func TestWaxCopperRejectsInvalidBlocks(t *testing.T) {
	for _, name := range []string{"copper_ore", "waxed_copper_block", "stone"} {
		if _, ok := WaxCopper(Block{Namespace: "minecraft", Name: name}); ok {
			t.Errorf("WaxCopper accepted %s", name)
		}
	}
}
