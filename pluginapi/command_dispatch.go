package pluginapi

import (
	"fmt"

	abi "GoCraft/abi/v1"
)

// invokeCommand runs one executor and reports what it asked for.
//
// The replies become chat.message effects rather than a result message of their
// own. The host queues them and delivers them on the tick, which is the one
// path a plugin has to the world — a command handler writing to a player
// directly would race the simulation reading it, exactly as an event handler
// would.
func (s *runtimeState) invokeCommand(invocation abi.CommandInvocation) abi.CommandResult {
	if !s.enabled || s.context == nil {
		return abi.CommandResult{Error: "pluginapi: plugin is not enabled"}
	}
	call, err := commandContextFrom(invocation)
	if err != nil {
		return abi.CommandResult{Error: err.Error()}
	}
	replies, callErr := s.context.commands.invoke(invocation.Executor, call)
	result := abi.CommandResult{}
	for _, reply := range replies {
		result.Effects = append(result.Effects, abi.HostCall{
			Type:   hostCallMessage,
			Fields: []abi.Value{invocation.Sender.Player, abi.String(reply)},
		})
	}
	// The replies travel even when the handler failed: one that refused and
	// said why has already queued that sentence, and dropping it would leave
	// the sender with nothing but a generic failure.
	if callErr != nil {
		result.Error = callErr.Error()
	}
	return result
}

// hostCallMessage is the host call that delivers a line to a player. Named here
// rather than imported from the host: pluginapi is compiled into plugin
// binaries, which must not link the server.
const hostCallMessage = "chat.message"

func commandContextFrom(invocation abi.CommandInvocation) (*CommandContext, error) {
	var sender *Player
	if invocation.Sender.Player.Kind != abi.ValueList || len(invocation.Sender.Player.List) != 0 {
		decoded, err := playerFrom(invocation.Sender.Player)
		if err != nil {
			return nil, err
		}
		sender = &decoded
	}
	arguments, err := commandValuesFrom(invocation.Arguments)
	if err != nil {
		return nil, err
	}
	return &CommandContext{
		Sender:      sender,
		SenderName:  invocation.Sender.Name,
		Args:        arguments,
		permissions: invocation.Sender,
	}, nil
}

func commandValuesFrom(arguments []abi.CommandArgument) (CommandValues, error) {
	values := make(CommandValues, len(arguments))
	for _, argument := range arguments {
		if _, duplicate := values[argument.Name]; duplicate {
			return nil, fmt.Errorf("pluginapi: duplicate command argument %s", argument.Name)
		}
		decoded, err := commandValueFrom(CommandValueKind(argument.Type), argument.Value)
		if err != nil {
			return nil, fmt.Errorf("pluginapi: command argument %s: %w", argument.Name, err)
		}
		values[argument.Name] = decoded
	}
	return values, nil
}
