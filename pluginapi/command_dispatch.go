package pluginapi

import (
	"fmt"
	"math"

	abi "GoCraft/abi/v1"
)

func (s *runtimeState) invokeCommand(event *abi.Event) (abi.Verdict, error) {
	if !s.enabled || s.context == nil {
		return abi.Verdict{}, fmt.Errorf("pluginapi: plugin is not enabled")
	}
	executor, call, err := commandContextFrom(event)
	if err != nil {
		return abi.Verdict{}, err
	}
	replies, callErr := s.context.commands.invoke(executor, call)
	verdict := abi.Verdict{}
	for _, reply := range replies {
		verdict.Effects = append(verdict.Effects, abi.HostCall{
			Type: abi.HostCallCommandReply, Fields: []abi.Value{abi.String(reply)},
		})
	}
	if callErr != nil {
		verdict.Effects = append(verdict.Effects, abi.HostCall{
			Type: abi.HostCallCommandFailed, Fields: []abi.Value{abi.String(callErr.Error())},
		})
	}
	return verdict, nil
}

func commandContextFrom(event *abi.Event) (uint32, *CommandContext, error) {
	if event == nil || len(event.Fields) != 4 {
		return 0, nil, fmt.Errorf("pluginapi: malformed command invocation")
	}
	executor := event.Fields[0]
	if executor.Kind != abi.ValueInt64 || executor.Int64 < 0 || executor.Int64 > math.MaxUint32 {
		return 0, nil, fmt.Errorf("pluginapi: invalid command executor")
	}
	var sender *Player
	if value := event.Fields[1]; value.Kind != abi.ValueList || len(value.List) != 0 {
		decoded, err := playerFrom(value)
		if err != nil {
			return 0, nil, err
		}
		sender = &decoded
	}
	name, err := stringFrom(event.Fields[2], "command sender name")
	if err != nil {
		return 0, nil, err
	}
	arguments, err := commandValuesFrom(event.Fields[3])
	if err != nil {
		return 0, nil, err
	}
	return uint32(executor.Int64), &CommandContext{Sender: sender, SenderName: name, Args: arguments}, nil
}

func commandValuesFrom(value abi.Value) (CommandValues, error) {
	if value.Kind != abi.ValueList {
		return nil, fmt.Errorf("pluginapi: command arguments are not a list")
	}
	values := make(CommandValues, len(value.List))
	for _, item := range value.List {
		entry, err := listOf(item, 3, "command argument")
		if err != nil {
			return nil, err
		}
		name, err := stringFrom(entry[0], "command argument name")
		if err != nil {
			return nil, err
		}
		if entry[1].Kind != abi.ValueInt64 {
			return nil, fmt.Errorf("pluginapi: command argument kind is not an integer")
		}
		if _, duplicate := values[name]; duplicate {
			return nil, fmt.Errorf("pluginapi: duplicate command argument %s", name)
		}
		decoded, err := commandValueFrom(CommandValueKind(entry[1].Int64), entry[2])
		if err != nil {
			return nil, err
		}
		values[name] = decoded
	}
	return values, nil
}
