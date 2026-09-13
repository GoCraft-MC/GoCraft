package server

import (
	"sort"
	"testing"
)

func TestMinecraft1214MobParityRegistryIsExhaustive(t *testing.T) {
	expected := []string{
		"minecraft:allay", "minecraft:armadillo", "minecraft:axolotl", "minecraft:bat",
		"minecraft:bee", "minecraft:blaze", "minecraft:bogged", "minecraft:breeze",
		"minecraft:camel", "minecraft:cat", "minecraft:cave_spider", "minecraft:chicken",
		"minecraft:cod", "minecraft:cow", "minecraft:creaking", "minecraft:creeper",
		"minecraft:dolphin", "minecraft:donkey", "minecraft:drowned", "minecraft:elder_guardian",
		"minecraft:ender_dragon", "minecraft:enderman", "minecraft:endermite", "minecraft:evoker",
		"minecraft:fox", "minecraft:frog", "minecraft:ghast", "minecraft:giant",
		"minecraft:glow_squid", "minecraft:goat", "minecraft:guardian", "minecraft:hoglin",
		"minecraft:horse", "minecraft:husk", "minecraft:illusioner", "minecraft:iron_golem",
		"minecraft:llama", "minecraft:magma_cube", "minecraft:mooshroom", "minecraft:mule",
		"minecraft:ocelot", "minecraft:panda", "minecraft:parrot", "minecraft:phantom",
		"minecraft:pig", "minecraft:piglin", "minecraft:piglin_brute", "minecraft:pillager",
		"minecraft:polar_bear", "minecraft:pufferfish", "minecraft:rabbit", "minecraft:ravager",
		"minecraft:salmon", "minecraft:sheep", "minecraft:shulker", "minecraft:silverfish",
		"minecraft:skeleton", "minecraft:skeleton_horse", "minecraft:slime", "minecraft:sniffer",
		"minecraft:snow_golem", "minecraft:spider", "minecraft:squid", "minecraft:stray",
		"minecraft:strider", "minecraft:tadpole", "minecraft:trader_llama", "minecraft:tropical_fish",
		"minecraft:turtle", "minecraft:vex", "minecraft:villager", "minecraft:vindicator",
		"minecraft:wandering_trader", "minecraft:warden", "minecraft:witch", "minecraft:wither",
		"minecraft:wither_skeleton", "minecraft:wolf", "minecraft:zoglin", "minecraft:zombie",
		"minecraft:zombie_horse", "minecraft:zombie_villager", "minecraft:zombified_piglin",
	}

	if len(expected) != 83 {
		t.Fatalf("test fixture must pin all 83 Java 1.21.4 mobs, got %d", len(expected))
	}
	if len(minecraft1214MobParitySpecs) != len(expected) {
		t.Fatalf("mob parity registry has %d entries, want %d", len(minecraft1214MobParitySpecs), len(expected))
	}

	actual := make([]string, 0, len(minecraft1214MobParitySpecs))
	seen := make(map[string]struct{}, len(minecraft1214MobParitySpecs))
	for _, spec := range minecraft1214MobParitySpecs {
		id := string(spec.Type)
		if id == "" || spec.Controller == "" || spec.Navigation == "" || spec.Temper == "" {
			t.Fatalf("incomplete parity spec: %+v", spec)
		}
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("duplicate mob parity entry %q", id)
		}
		seen[id] = struct{}{}
		actual = append(actual, id)
	}

	sort.Strings(actual)
	sort.Strings(expected)
	for index := range expected {
		if actual[index] != expected[index] {
			t.Fatalf("mob parity registry mismatch at %d: got %q want %q", index, actual[index], expected[index])
		}
	}
}

func TestVillagerParityRemainsDedicated(t *testing.T) {
	for _, spec := range minecraft1214MobParitySpecs {
		if string(spec.Type) == "minecraft:villager" {
			if spec.Controller != "villager-brain" {
				t.Fatalf("villager must remain on the dedicated Brain implementation, got %q", spec.Controller)
			}
			return
		}
	}
	t.Fatal("villager missing from parity registry")
}
