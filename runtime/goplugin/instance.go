package goplugin

import (
	"context"
	"errors"
	"sync"

	abi "GoCraft/abi/v1"
	"GoCraft/core/plugin"
	"GoCraft/runtime/ipc"
)

// Instance is one native plugin process.
type Instance struct {
	runtime    *Runtime
	manifest   plugin.Manifest
	supervisor *ipc.Supervisor
	cleanup    func()
	unloadOnce sync.Once
	unloadErr  error
}

func (i *Instance) Manifest() plugin.Manifest { return i.manifest }

func (i *Instance) Dispatch(ctx context.Context, event *abi.Event) (abi.Verdict, error) {
	return i.supervisor.Dispatch(ctx, i.manifest.ID, event)
}

func (i *Instance) Unload(ctx context.Context) error {
	i.unloadOnce.Do(func() {
		unloadErr := i.supervisor.Unload(i.manifest.ID)
		stopErr := i.supervisor.Stop(ctx)
		if i.cleanup != nil {
			i.cleanup()
		}
		i.runtime.remove(i.manifest.ID)
		i.unloadErr = errors.Join(unloadErr, stopErr)
	})
	return i.unloadErr
}
