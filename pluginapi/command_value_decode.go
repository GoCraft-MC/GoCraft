package pluginapi

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
		item, err := stringFrom(value, "item argument")
		if err != nil {
			return decoded, err
		}
		decoded.Item = item
	case CommandDuration:
		if value.Kind != abi.ValueInt64 {
			return decoded, wrongCommandValue(kind)
		}
		decoded.Duration = time.Duration(value.Int64)
	default:
		return decoded, fmt.Errorf("pluginapi: unsupported command value kind %d", kind)
	}
	return decoded, nil
}

func wrongCommandValue(kind CommandValueKind) error {
	return fmt.Errorf("pluginapi: invalid value for command kind %d", kind)
}
