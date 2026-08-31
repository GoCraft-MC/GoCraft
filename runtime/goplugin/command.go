package goplugin

import (
	"context"
	"errors"
	"fmt"

	abi "GoCraft/abi/v1"
	"GoCraft/core/command"
)

func (i *Instance) InvokeCommand(ctx context.Context, executor command.ExecID,
	sender command.Sender, values command.Values) error {
	if sender == nil {
		return fmt.Errorf("go runtime: command sender is required")
	}
	event, err := commandEvent(executor, sender, values)
	if err != nil {
		return err
	}
	verdict, err := i.supervisor.Dispatch(ctx, i.manifest.ID, event)
	if err != nil {
		return err
	}
	var joined error
	for _, effect := range verdict.Effects {
		if len(effect.Fields) != 1 || effect.Fields[0].Kind != abi.ValueString {
			joined = errors.Join(joined, fmt.Errorf("go runtime: malformed command result %s", effect.Type))
			continue
		}
		switch effect.Type {
		case abi.HostCallCommandReply:
			if err := sender.SendMessage(effect.Fields[0].String); err != nil {
				joined = errors.Join(joined, err)
			}
		case abi.HostCallCommandFailed:
			joined = errors.Join(joined, errors.New(effect.Fields[0].String))
		default:
			joined = errors.Join(joined, fmt.Errorf("go runtime: unknown command result %s", effect.Type))
		}
	}
	return joined
}
