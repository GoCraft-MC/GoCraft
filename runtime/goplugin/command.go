package goplugin

import (
	"context"
	"fmt"

	"GoCraft/core/dispatch"
	"GoCraft/core/plugin"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/command"
)

// InvokeCommand runs one of the plugin's command executors in its process.
//
// The conversion is core/plugin's, not this package's. Writing it here would be
// the same conversion written once per runtime — which is how this ended up
// encoding a duration in nanoseconds while the JVM encoded milliseconds, from
// two functions that were otherwise line-for-line the same.
//
// Effects come back rather than being applied here. A reply travels as a
// chat.message the tick delivers, exactly as a verdict's effects do, so a
// command handler and an event handler reach the world by one path.
func (i *Instance) InvokeCommand(ctx context.Context, executor command.ExecID,
	sender dispatch.Sender, values dispatch.Values) (abi.CommandResult, error) {
	if sender == nil {
		return abi.CommandResult{}, fmt.Errorf("go runtime: command sender is required")
	}
	invocation, err := plugin.NewCommandInvocation(executor, sender, values, i.manifest.Permissions)
	if err != nil {
		return abi.CommandResult{}, err
	}
	return i.supervisor.Invoke(ctx, i.manifest.ID, invocation)
}
