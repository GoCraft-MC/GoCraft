package plugin

import (
	"fmt"
	"sort"

	"GoCraft/core/dispatch"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/command"
)

// NewCommandInvocation converts one parsed command into the neutral form an
// out-of-process runtime is sent.
//
// It lives here rather than in a runtime package because every backend crosses
// the boundary the same way, and a conversion written once per language is the
// same conversion free to drift. The vocabulary types it produces are the ones
// events already use, for the same reason: a PlayerRef read out of a command
// argument and one read out of an event have to be the same shape or a handler
// needs two readers for one concept.
//
// permissions is what the plugin's manifest declared. Every node is resolved
// here because the ABI has no message for asking later — the same gap events
// work around, and cheap to over-supply at typing speed.
func NewCommandInvocation(
	executor command.ExecID,
	sender dispatch.Sender,
	arguments dispatch.Values,
	permissions []string,
) (abi.CommandInvocation, error) {
	converted, err := commandArguments(arguments)
	if err != nil {
		return abi.CommandInvocation{}, err
	}
	return abi.CommandInvocation{
		Executor:  uint32(executor),
		Sender:    commandSender(sender, permissions),
		Arguments: converted,
	}, nil
}

// commandSender resolves the sender once, up front.
//
// A nil sender is the console in every way that matters here: no player, no
// name, and no permission it holds. It is not an error — the registry already
// refused the invocation if the path needed one.
func commandSender(sender dispatch.Sender, permissions []string) abi.CommandSender {
	resolved := abi.CommandSender{Player: playerReference(nil)}
	if sender != nil {
		player, _ := sender.Player()
		resolved.Player = playerReference(player)
		resolved.Name = sender.Name()
	}
	// Sorted and deduplicated so the payload is byte-identical for the same
	// inputs, whatever order the manifest happened to list them in.
	nodes := make([]string, 0, len(permissions))
	seen := make(map[string]struct{}, len(permissions))
	for _, node := range permissions {
		if _, done := seen[node]; done {
			continue
		}
		seen[node] = struct{}{}
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		allowed := sender != nil && sender.Has(node)
		resolved.Permissions = append(resolved.Permissions,
			abi.List(abi.String(node), abi.Bool(allowed)))
	}
	return resolved
}

// commandArguments converts the parsed arguments, in name order.
//
// Values is a map, so without sorting the same command would serialise
// differently from one invocation to the next. Nothing depends on the order —
// the arguments carry their names — but a payload that varies for identical
// input is one nobody can compare in a log or a test.
func commandArguments(arguments dispatch.Values) ([]abi.CommandArgument, error) {
	names := make([]string, 0, len(arguments))
	for name := range arguments {
		names = append(names, name)
	}
	sort.Strings(names)

	converted := make([]abi.CommandArgument, 0, len(names))
	for _, name := range names {
		argument := arguments[name]
		kind, err := commandArgumentType(argument.Type)
		if err != nil {
			return nil, fmt.Errorf("command argument %s: %w", name, err)
		}
		value, err := commandArgumentValue(argument)
		if err != nil {
			return nil, fmt.Errorf("command argument %s: %w", name, err)
		}
		converted = append(converted, abi.CommandArgument{Name: name, Type: kind, Value: value})
	}
	return converted, nil
}

func commandArgumentType(kind command.ArgType) (abi.CommandArgumentType, error) {
	switch kind {
	case command.ArgInteger:
		return abi.CommandArgumentInteger, nil
	case command.ArgDecimal:
		return abi.CommandArgumentDecimal, nil
	case command.ArgString:
		return abi.CommandArgumentString, nil
	case command.ArgGreedy:
		return abi.CommandArgumentGreedy, nil
	case command.ArgPlayer:
		return abi.CommandArgumentPlayer, nil
	case command.ArgBlockPos:
		return abi.CommandArgumentBlockPos, nil
	case command.ArgBlockState:
		return abi.CommandArgumentBlockState, nil
	case command.ArgItem:
		return abi.CommandArgumentItem, nil
	case command.ArgDuration:
		return abi.CommandArgumentDuration, nil
	case command.ArgEnum:
		return abi.CommandArgumentEnum, nil
	case command.ArgCustom:
		return abi.CommandArgumentCustom, nil
	default:
		return abi.CommandArgumentInvalid, fmt.Errorf("unknown type %d", kind)
	}
}

// commandArgumentValue picks the field of Value the type selected.
//
// Value carries every parsed shape at once and only one of them is meaningful,
// so reading the wrong field yields a zero rather than a failure. Switching on
// the declared type is what keeps that from crossing the socket unnoticed.
func commandArgumentValue(argument dispatch.Value) (abi.Value, error) {
	switch argument.Type {
	case command.ArgInteger:
		return abi.Int64(argument.Integer), nil
	case command.ArgDecimal:
		return abi.Double(argument.Decimal), nil
	case command.ArgString, command.ArgGreedy, command.ArgEnum, command.ArgCustom:
		return abi.String(argument.String), nil
	case command.ArgPlayer:
		return playerReference(argument.Player), nil
	case command.ArgBlockPos:
		return positionValue(argument.Position), nil
	case command.ArgBlockState:
		return blockValue(argument.Block), nil
	case command.ArgItem:
		return itemValue(argument.Item), nil
	case command.ArgDuration:
		// Milliseconds, not a Duration: nanoseconds are Go's unit and no other
		// runtime shares it, and a tick is 50 ms, so nothing here is finer.
		return abi.Int64(argument.Duration.Milliseconds()), nil
	default:
		return abi.Value{}, fmt.Errorf("unknown type %d", argument.Type)
	}
}
