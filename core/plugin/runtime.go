package plugin

import (
	"context"

	abi "GoCraft/abi/v1"
	"GoCraft/core/command"
)

// Host is the only mutation path exposed to a plugin runtime.
type Host interface {
	Enqueue(call abi.HostCall) error
}

// Artifact describes a pinned runtime or library download.
type Artifact struct {
	URL    string
	SHA256 string
	Size   int64
	Strip  int
	Bin    string
}

// Provisioner supplies content-addressed artifacts during boot preflight.
type Provisioner interface {
	Cached(key string) (path string, ok bool)
	Fetch(ctx context.Context, key string, artifact Artifact) (string, error)
}

// Runtime hosts all plugins using one language backend.
type Runtime interface {
	Name() string
	Provision(ctx context.Context, provisioner Provisioner) error
	Start(ctx context.Context, host Host) error
	Load(ctx context.Context, bundle Bundle) (Instance, error)
	Stop(ctx context.Context) error
}

// Instance is one loaded plugin, regardless of its runtime.
type Instance interface {
	Manifest() Manifest
	Dispatch(ctx context.Context, event *abi.Event) (abi.Verdict, error)
	Unload(ctx context.Context) error
}

// CommandInstance is implemented by runtimes that loaded a command tree.
type CommandInstance interface {
	InvokeCommand(context.Context, command.ExecID, command.Sender, command.Values) error
}
