package goplugin

import (
	"sync"

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
