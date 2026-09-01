package goplugin

import (
	"context"
	"errors"
	"sync"

	"GoCraft/runtime/link"
	abi "github.com/GoCraft-MC/gocraft-abi/abi/v1"
	"github.com/GoCraft-MC/gocraft-abi/gcpkg"
)

// Instance is one native plugin process.
type Instance struct {
	runtime    *Runtime
	manifest   gcpkg.Manifest
	supervisor *link.Supervisor
	cleanup    func()
	unloadOnce sync.Once
	unloadErr  error
}

func (i *Instance) Manifest() gcpkg.Manifest { return i.manifest }

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
