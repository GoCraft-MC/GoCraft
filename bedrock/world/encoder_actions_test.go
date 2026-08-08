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
