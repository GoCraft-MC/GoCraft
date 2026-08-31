package gocraft

import (
	"fmt"
	"time"

	abi "GoCraft/abi/v1"
)

func commandValueFrom(kind CommandValueKind, value abi.Value) (CommandValue, error) {
	decoded := CommandValue{Kind: kind}
	switch kind {
	case CommandInteger:
		if value.Kind != abi.ValueInt64 {
			return decoded, wrongCommandValue(kind)
		}
		decoded.Integer = value.Int64
	case CommandDecimal:
		if value.Kind != abi.ValueDouble {
			return decoded, wrongCommandValue(kind)
		}
		decoded.Decimal = value.Double
	case CommandString, CommandGreedy, CommandEnum, CommandCustom:
		text, err := stringFrom(value, "command argument")
		if err != nil {
			return decoded, err
		}
		decoded.Text = text
	case CommandPlayer:
		player, err := playerFrom(value)
		if err != nil {
			return decoded, err
		}
		decoded.Player = &player
	case CommandBlockPos:
		position, err := positionFrom(value)
		if err != nil {
			return decoded, err
		}
		decoded.Position = position
	case CommandBlockState:
		block, err := blockFrom(value)
		if err != nil {
			return decoded, err
		}
		decoded.Block = block
	case CommandItem:
		item, err := itemFrom(value)
		if err != nil {
			return decoded, err
		}
		decoded.Item = item
	case CommandDuration:
		if value.Kind != abi.ValueInt64 {
			return decoded, wrongCommandValue(kind)
		}
		// Milliseconds on the wire. No two runtimes agree on a finer unit, and
		// a tick is fifty of them.
		decoded.Duration = time.Duration(value.Int64) * time.Millisecond
	default:
		return decoded, fmt.Errorf("gocraft: unsupported command value kind %d", kind)
	}
	return decoded, nil
}

// itemFrom reads the ItemRef vocabulary type: id, count, damage.
func itemFrom(value abi.Value) (Item, error) {
	entry, err := listOf(value, 3, "item argument")
	if err != nil {
		return Item{}, err
	}
	id, err := stringFrom(entry[0], "item id")
	if err != nil {
		return Item{}, err
	}
	if entry[1].Kind != abi.ValueInt64 || entry[2].Kind != abi.ValueInt64 {
		return Item{}, fmt.Errorf("gocraft: item count and damage are not integers")
	}
	return Item{ID: id, Count: entry[1].Int64, Damage: entry[2].Int64}, nil
}

func wrongCommandValue(kind CommandValueKind) error {
	return fmt.Errorf("gocraft: invalid value for command kind %d", kind)
}
