// Package jvm drives the Java plugin runtime. It is Go, and it contains no
// Java: the jar it spawns is built in the gocraft-java repository, so the Go
// repository never needs a JDK to build or release.
//
// Everything below the process boundary — framing, correlation, liveness, the
// abi/wire conversion — belongs to runtime/ipc and is not repeated here. What
// is left is the part only Java has: finding a JDK, carrying the jar, and
// building the command line. That is the whole of what a runtime package is
// supposed to be.
package jvm

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	abi "GoCraft/abi/v1"
	"GoCraft/core/plugin"
	"GoCraft/runtime/ipc"
)

// RuntimeName is what a plugin manifest writes in its runtime field.
const RuntimeName = "jvm"

// abiVersion is the ABI the host speaks. A runtime announcing anything else is
// refused during the handshake rather than negotiated with.
const abiVersion = 1

// Config is everything the Java runtime needs that is not derivable.
type Config struct {
	// JavaPath forces a specific java binary and outranks JAVA_HOME and PATH.
	// Empty means detect.
	JavaPath string

	// PreferSystem keeps an existing JDK that is new enough rather than
	// downloading a second copy. Provisioning is still what happens when none
	// is found.
	PreferSystem bool

	// JarPath runs a runtime jar from disk instead of the embedded one, which
	// is what a developer working against a gocraft-java checkout wants.
	JarPath string

	// ExtractDirectory holds extracted jars. Empty picks the user cache
	// directory, then the temporary directory.
	ExtractDirectory string

	// SocketDirectory holds the Unix socket. It must be short: the whole path
	// has a 107 byte budget, and the runtime name and process id spend part
	// of it.
	SocketDirectory string

	TickRate     uint32
	EventBudget  time.Duration
	StartTimeout time.Duration
	Liveness     ipc.Liveness

	// Stdout and Stderr receive the JVM's own output. Empty means the server's,
	// so a stack trace from a plugin lands in the console and in latest.log
	// beside everything else.
	Stdout io.Writer
	Stderr io.Writer

	// Probe reports the major Java version of a candidate binary. Production
	// leaves it nil and gets `java -version`; a test supplies its own so the Go
	// repository's suite never needs a JVM installed to run.
	Probe func(ctx context.Context, java string) (int, error)
}

// Runtime hosts every Java plugin in one child process.
//
// It satisfies plugin.Runtime without saying so: Go interfaces are structural,
// and core/plugin must not be imported here for the declaration's sake alone —
// the dependency runs the other way.
type Runtime struct {
	config     Config
	supervisor *ipc.Supervisor

	mu   sync.RWMutex
	java string
	host plugin.Host
}

// New prepares the runtime. Nothing is spawned and nothing is downloaded until
// Provision and Start are called, so registering it costs nothing on a server
// that has no Java plugin — which is what makes "no Java runtime is required"
// true rather than aspirational.
func New(config Config) *Runtime {
	if config.StartTimeout <= 0 {
		config.StartTimeout = 60 * time.Second
	}
	return &Runtime{config: config}
}

func (r *Runtime) Name() string { return RuntimeName }

// Java is the binary Provision settled on, empty before it ran.
func (r *Runtime) Java() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.java
}

func (r *Runtime) setJava(java string) {
	r.mu.Lock()
	r.java = java
	r.mu.Unlock()
}

// Start spawns the JVM and completes the handshake. It returns only once the
// runtime is ready to be sent plugins, because the listeners are waiting on it:
// a server accepting players while a spawn-protection plugin is still loading
// is worse than a server that takes longer to boot.
func (r *Runtime) Start(ctx context.Context, host plugin.Host) error {
	java := r.Java()
	if java == "" {
		return fmt.Errorf("jvm: Start before Provision found a JDK")
	}
	jar, err := r.ensureJar()
	if err != nil {
		return err
	}

	r.mu.Lock()
	// Held rather than used. The mutation queue already receives the effects a
	// verdict carries, and the schema has no unsolicited host call yet; when it
	// gains one this is where it arrives, and re-plumbing it then would be
	// churn for nothing.
	r.host = host
	r.mu.Unlock()

	supervisor := ipc.NewSupervisor(ipc.Config{
		Runtime:      RuntimeName,
		Directory:    r.socketDirectory(),
		ABI:          abiVersion,
		TickRate:     r.config.TickRate,
		EventBudget:  r.config.EventBudget,
		StartTimeout: r.config.StartTimeout,
		Spawn:        r.spawn(java, jar),
	}, r.config.Liveness)

	if err := supervisor.Start(ctx); err != nil {
		return err
	}
	r.mu.Lock()
	r.supervisor = supervisor
	r.mu.Unlock()
	return nil
}

// spawn builds the command line, and is the only thing in this package that
// runtime/python or runtime/go would write differently. Everything else they
// share through ipc.
func (r *Runtime) spawn(java, jar string) ipc.Spawn {
	return func(socket string) *exec.Cmd {
		command := exec.Command(java,
			// protobuf reaches for sun.misc.Unsafe, which Java 24 made a
			// terminal deprecation: without this the JVM prints four warning
			// lines into the server console and latest.log on every single
			// boot. Silencing it is right because the call is protobuf's and
			// not something an admin can act on — and a warning that appears
			// unconditionally is one nobody reads when it matters.
			"--sun-misc-unsafe-memory-access=allow",
			"-jar", jar,
			"--sock", socket,
			"--abi", strconv.Itoa(abiVersion),
		)
		command.Stdout, command.Stderr = r.config.Stdout, r.config.Stderr
		if command.Stdout == nil {
			command.Stdout = os.Stdout
		}
		if command.Stderr == nil {
			command.Stderr = os.Stderr
		}
		return command
	}
}

func (r *Runtime) socketDirectory() string {
	if r.config.SocketDirectory != "" {
		return r.config.SocketDirectory
	}
	return os.TempDir()
}

// Load brings up one plugin inside the running JVM.
func (r *Runtime) Load(ctx context.Context, bundle plugin.Bundle) (plugin.Instance, error) {
	// Checked before the process is, because it is a property of the bundle
	// rather than of the runtime's state: the answer is the same whether the
	// JVM is up or not, and it is the more useful of the two messages.
	//
	// A command tree cannot be honoured yet — the envelope reserves the fields
	// for command invocation and does not define them, so there is no way to
	// reach a plugin's executor across the process boundary. Loading anyway
	// would give an admin a plugin whose commands silently do nothing.
	if bundle.Commands != nil {
		return nil, fmt.Errorf(
			"jvm: plugin %s declares commands, which the ABI cannot yet carry to an "+
				"out-of-process runtime", bundle.Manifest.ID)
	}
	supervisor, err := r.running()
	if err != nil {
		return nil, err
	}

	events := make([]string, 0, len(bundle.Manifest.Subscriptions))
	for _, subscription := range bundle.Manifest.Subscriptions {
		events = append(events, subscription.Event)
	}
	if _, err := supervisor.Load(ctx, ipc.LoadRequest{
		ID:         bundle.Manifest.ID,
		BundlePath: bundle.Path,
		Entry:      bundle.Manifest.Entry,
		Events:     events,
	}); err != nil {
		return nil, err
	}
	return &Instance{supervisor: supervisor, manifest: bundle.Manifest}, nil
}

// Ready ends the load phase. The host opens its listeners only after it.
func (r *Runtime) Ready(context.Context) error {
	supervisor, err := r.running()
	if err != nil {
		return err
	}
	return supervisor.Ready()
}

// Stop asks the JVM to leave and makes sure it did. It is safe on a runtime
// that never started, which is how a rolled-back boot unwinds.
func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	supervisor := r.supervisor
	r.supervisor = nil
	r.mu.Unlock()
	if supervisor == nil {
		return nil
	}
	return supervisor.Stop(ctx)
}

// Failed is closed when the JVM dies on its own, and stays open across a Stop
// the host asked for. Respawn is not implemented; this is the signal it will
// hang off when it is.
func (r *Runtime) Failed() <-chan struct{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.supervisor == nil {
		return nil
	}
	return r.supervisor.Failed()
}

func (r *Runtime) running() (*ipc.Supervisor, error) {
	r.mu.RLock()
	supervisor := r.supervisor
	r.mu.RUnlock()
	if supervisor == nil {
		return nil, fmt.Errorf("jvm: the runtime is not running")
	}
	return supervisor, nil
}

// Instance is one Java plugin. Every method is a forward: the JVM holds the
// plugin, this holds nothing but its name.
type Instance struct {
	supervisor *ipc.Supervisor
	manifest   plugin.Manifest
}

func (i *Instance) Manifest() plugin.Manifest { return i.manifest }

func (i *Instance) Dispatch(ctx context.Context, event *abi.Event) (abi.Verdict, error) {
	return i.supervisor.Dispatch(ctx, i.manifest.ID, event)
}

// Unload drops the plugin. The context is unused because the schema carries no
// acknowledgement for UNLOAD — there is nothing to wait for, and pretending
// otherwise would put a timeout on a message that is already gone.
func (i *Instance) Unload(context.Context) error {
	return i.supervisor.Unload(i.manifest.ID)
}
