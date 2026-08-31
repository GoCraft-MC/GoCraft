package goplugin

import (
	"fmt"
	"sort"

	abi "GoCraft/abi/v1"
	"GoCraft/core/command"
)

func commandEvent(executor command.ExecID, sender command.Sender, values command.Values) (*abi.Event, error) {
	playerRef, _ := sender.Player()
	arguments, err := commandArguments(values)
	if err != nil {
		return nil, err
	}
	return &abi.Event{
		Type:      abi.EventCommandInvoke,
		OnFailure: abi.FailureAllow,
		Fields: []abi.Value{
			abi.Int64(int64(executor)),
			playerValue(playerRef),
			abi.String(sender.Name()),
			abi.List(arguments...),
		},
	}, nil
}

func commandArguments(values command.Values) ([]abi.Value, error) {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	arguments := make([]abi.Value, 0, len(names))
	for _, name := range names {
		value := values[name]
		encoded, err := commandArgument(value)
		if err != nil {
			return nil, fmt.Errorf("command argument %s: %w", name, err)
		}
		arguments = append(arguments, abi.List(
			abi.String(name), abi.Int64(int64(value.Type)), encoded))
	}
	return arguments, nil
}

func commandArgument(value command.Value) (abi.Value, error) {
	switch value.Type {
	case command.ArgInteger:
		return abi.Int64(value.Integer), nil
	case command.ArgDecimal:
		return abi.Double(value.Decimal), nil
	case command.ArgString, command.ArgGreedy, command.ArgEnum, command.ArgCustom:
		return abi.String(value.String), nil
	case command.ArgPlayer:
		return playerValue(value.Player), nil
	case command.ArgBlockPos:
		return positionValue(value.Position), nil
	case command.ArgBlockState:
		return blockValue(value.Block), nil
	case command.ArgItem:
		return abi.String(value.Item.ItemID), nil
	case command.ArgDuration:
		return abi.Int64(int64(value.Duration)), nil
	default:
		return abi.Value{}, fmt.Errorf("unsupported type %d", value.Type)
	}
}
