package world

import "testing"

func TestAddCandleToCake(t *testing.T) {
	cake := Block{Namespace: "minecraft", Name: "cake", Properties: map[string]string{"bites": "0"}}
	for item, want := range map[string]string{
		"minecraft:candle":      "minecraft:candle_cake",
		"minecraft:blue_candle": "minecraft:blue_candle_cake",
	} {
		block, ok := AddCandleToCake(cake, item)
		if !ok || block.ResourceLocation() != want || block.Properties["lit"] != "false" {
			t.Errorf("AddCandleToCake(%s) = %+v, %v", item, block, ok)
		}
	}
	if _, ok := AddCandleToCake(cake, "minecraft:torch"); ok {
		t.Fatal("cake accepted a non-candle item")
	}
}
