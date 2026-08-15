package world

import (
	"fmt"
	"testing"

	coreworld "GoCraft/core/world"
)

func TestActionBlockStatesTranslateToBedrockPalette(t *testing.T) {
	encoder := NewEncoder()
	tests := []struct {
		name      string
		block     coreworld.Block
		wantName  string
		wantState string
		wantValue any
	}{
		{
			name: "floor torch", block: coreworld.Block{Namespace: "minecraft", Name: "torch", Properties: map[string]string{"facing": "up"}},
			wantName: "minecraft:torch", wantState: "torch_facing_direction", wantValue: "top",
		},
		{
			name: "wall torch", block: coreworld.Block{Namespace: "minecraft", Name: "wall_torch", Properties: map[string]string{"facing": "north"}},
			wantName: "minecraft:torch", wantState: "torch_facing_direction", wantValue: "north",
		},
		{
			name: "powered lever", block: coreworld.Block{Namespace: "minecraft", Name: "lever", Properties: map[string]string{"face": "wall", "facing": "north", "powered": "true"}},
			wantName: "minecraft:lever", wantState: "open_bit", wantValue: uint8(1),
		},
		{
			name: "redstone power", block: coreworld.Block{Namespace: "minecraft", Name: "redstone_wire", Properties: map[string]string{"power": "12"}},
			wantName: "minecraft:redstone_wire", wantState: "redstone_signal", wantValue: int32(12),
		},
		{
			name: "invisible light", block: coreworld.Block{Namespace: "minecraft", Name: "light", Properties: map[string]string{"level": "9"}},
			wantName: "minecraft:light_block_9",
		},
		{
			name: "snow layers", block: coreworld.Block{Namespace: "minecraft", Name: "snow", Properties: map[string]string{"layers": "3"}},
			wantName: "minecraft:snow_layer", wantState: "height", wantValue: int32(2),
		},
		{
			name: "four candles", block: coreworld.Block{Namespace: "minecraft", Name: "candle", Properties: map[string]string{"candles": "4", "lit": "false"}},
			wantName: "minecraft:candle", wantState: "candles", wantValue: int32(3),
		},
		{
			name: "ready composter", block: coreworld.Block{Namespace: "minecraft", Name: "composter", Properties: map[string]string{"level": "8"}},
			wantName: "minecraft:composter", wantState: "composter_fill_level", wantValue: int32(8),
		},
		{
			name: "spruce door facing", block: coreworld.Block{Namespace: "minecraft", Name: "spruce_door", Properties: map[string]string{
				"facing": "north", "half": "lower", "hinge": "left", "open": "true", "powered": "false",
			}},
			wantName: "minecraft:spruce_door", wantState: "minecraft:cardinal_direction", wantValue: "east",
		},
		{
			name: "open unpowered fence gate", block: coreworld.Block{Namespace: "minecraft", Name: "spruce_fence_gate", Properties: map[string]string{
				"facing": "north", "in_wall": "false", "open": "true", "powered": "false",
			}},
			wantName: "minecraft:spruce_fence_gate", wantState: "open_bit", wantValue: uint8(1),
		},
		{
			name: "connected wall", block: coreworld.Block{Namespace: "minecraft", Name: "cobblestone_wall", Properties: map[string]string{
				"north": "none", "east": "low", "south": "none", "west": "none", "up": "true", "waterlogged": "false",
			}},
			wantName: "minecraft:cobblestone_wall", wantState: "wall_connection_type_east", wantValue: "short",
		},
		{
			name: "nether portal", block: coreworld.Block{Namespace: "minecraft", Name: "nether_portal", Properties: map[string]string{"axis": "x"}},
			wantName: "minecraft:portal", wantState: "portal_axis", wantValue: "x",
		},
		{
			name: "filled end portal frame", block: coreworld.Block{Namespace: "minecraft", Name: "end_portal_frame", Properties: map[string]string{
				"facing": "south", "eye": "true",
			}},
			wantName: "minecraft:end_portal_frame", wantState: "end_portal_eye_bit", wantValue: uint8(1),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, states := encoder.resolveState(test.block)
			if name != test.wantName {
				t.Fatalf("name = %q, want %q (states=%v)", name, test.wantName, states)
			}
			if test.wantState != "" && fmt.Sprint(states[test.wantState]) != fmt.Sprint(test.wantValue) {
				t.Fatalf("%s = %v, want %v (all states=%v)", test.wantState, states[test.wantState], test.wantValue, states)
			}
		})
	}
}

func TestDoorHalvesResolveToDistinctSynchronizedBedrockStates(t *testing.T) {
	encoder := NewEncoder()
	base := coreworld.Block{Namespace: "minecraft", Name: "spruce_door", Properties: map[string]string{
		"facing": "north", "half": "lower", "hinge": "right", "open": "true", "powered": "false",
	}}
	lowerName, lower := encoder.resolveState(base)
	upperBlock := base
	upperBlock.Properties = map[string]string{
		"facing": "north", "half": "upper", "hinge": "right", "open": "true", "powered": "false",
	}
	upperName, upper := encoder.resolveState(upperBlock)
	if lowerName != "minecraft:spruce_door" || upperName != lowerName {
		t.Fatalf("door names = lower %q upper %q", lowerName, upperName)
	}
	if fmt.Sprint(lower["upper_block_bit"]) != "0" || fmt.Sprint(upper["upper_block_bit"]) != "1" {
		t.Fatalf("upper bits = lower %v upper %v", lower["upper_block_bit"], upper["upper_block_bit"])
	}
	for _, states := range []map[string]any{lower, upper} {
		if fmt.Sprint(states["open_bit"]) != "1" || fmt.Sprint(states["door_hinge_bit"]) != "1" {
			t.Fatalf("door state lost open/hinge synchronization: %v", states)
		}
	}
	if encoder.BlockNetworkID(base) == encoder.BlockNetworkID(upperBlock) {
		t.Fatal("lower and upper door halves resolved to the same Bedrock network hash")
	}
}
