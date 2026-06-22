package main

import "testing"

func TestParseConstantsAcceptsPumpkinDataDrivenFormatting(t *testing.T) {
	items := parseConstants(`
pub const BUNDLE : Self = Self {
    id : 860 ,
    registry_key : "minecraft:bundle" ,
};
pub const IRON_PICKAXE: Self = Self {
    id: 299,
    registry_key: "minecraft:iron_pickaxe",
};`)

	for constant, want := range map[string]itemDefinition{
		"BUNDLE":       {registryKey: "minecraft:bundle", runtimeID: 860},
		"IRON_PICKAXE": {registryKey: "minecraft:iron_pickaxe", runtimeID: 299},
	} {
		if got, ok := items[constant]; !ok || got != want {
			t.Errorf("%s = %+v, %v; want %+v, true", constant, got, ok, want)
		}
	}
}
