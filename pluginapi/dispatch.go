package pluginapi

import (
	"fmt"

	abi "GoCraft/abi/v1"
)

func (s *runtimeState) dispatch(incoming *abi.Event) (abi.Verdict, error) {
	if !s.enabled || s.context == nil {
		return abi.Verdict{}, fmt.Errorf("pluginapi: plugin is not enabled")
	}
	event, err := eventFrom(incoming)
	if err != nil {
		return abi.Verdict{}, err
	}
	s.context.events.dispatch(event)
	verdict := abi.Verdict{}
	if cancellable, ok := event.(CancellableEvent); ok {
		verdict.Cancelled = cancellable.Cancelled()
	}
	return verdict, nil
}

func eventFrom(incoming *abi.Event) (Event, error) {
	if incoming == nil {
		return nil, fmt.Errorf("pluginapi: missing event")
	}
	switch incoming.Type {
	case EventPlayerJoin:
		return playerJoinFrom(incoming.Fields)
	case EventBlockBreak:
		return blockBreakFrom(incoming.Fields)
	default:
		return nil, fmt.Errorf("pluginapi: unsupported event %s", incoming.Type)
	}
}

func playerJoinFrom(fields []abi.Value) (*PlayerJoinEvent, error) {
	if len(fields) != 2 {
		return nil, fmt.Errorf("pluginapi: player.join has %d fields, want 2", len(fields))
	}
	player, err := playerFrom(fields[0])
	if err != nil {
		return nil, err
	}
	permissions, err := permissionsFrom(fields[1])
	if err != nil {
		return nil, err
	}
	return &PlayerJoinEvent{Player: player, Permissions: permissions}, nil
}

func blockBreakFrom(fields []abi.Value) (*BlockBreakEvent, error) {
	if len(fields) != 5 {
		return nil, fmt.Errorf("pluginapi: block.break has %d fields, want 5", len(fields))
	}
	player, err := playerFrom(fields[0])
	if err != nil {
		return nil, err
	}
	position, err := positionFrom(fields[1])
	if err != nil {
		return nil, err
	}
	block, err := blockFrom(fields[2])
	if err != nil {
		return nil, err
	}
	tool, err := stringFrom(fields[3], "block break tool")
	if err != nil {
		return nil, err
	}
	permissions, err := permissionsFrom(fields[4])
	if err != nil {
		return nil, err
	}
	return &BlockBreakEvent{Player: player, Position: position, Block: block,
		Tool: tool, Permissions: permissions}, nil
}
