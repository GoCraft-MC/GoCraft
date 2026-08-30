package ipc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	abi "GoCraft/abi/v1"
	wire "GoCraft/abi/v1/wire"
)

// Supervisor is the whole of what an out-of-process runtime shares with every
// other one: bring the process up, load plugins into it, dispatch events, take
// it back down, and notice when it dies in between.
//
// It exists so that a runtime package is delegation and nothing else. §14 puts
// the number at roughly 150 lines for a complete backend, and the only way that
// holds is if the transport, the correlation and the abi ↔ wire conversion are
// written once here rather than once per language. runtime/jvm, a future
// runtime/go and a future runtime/python differ in how a process is started —
// which is the Spawn function in Config — and in nothing else.
//
// It deliberately speaks abi types outward and wire types inward. The generated
// types stop at this file and convert.go: a runtime package that had to encode
// its own envelopes would be the second copy of a conversion that must not
// drift, which is the whole reason this type exists.
//
// The supervisor does not implement plugin.Runtime, and this package does not
// import core/plugin. The dependency only makes sense in one direction: the
// host knows about transports, a transport must not know about the registry
// that drives it. A runtime package embeds a Supervisor and satisfies
// plugin.Runtime itself, which is also what keeps Bundle out of here.
//
// One supervisor drives one process. A runtime that hosts N plugins in one
// process — the JVM — holds a single supervisor; a runtime whose plugins are
// each their own binary holds one per plugin.
type Supervisor struct {
	config   Config
	liveness Liveness

	mu          sync.RWMutex
	child       *Child
	cancelWatch context.CancelFunc
	failed      chan struct{}
	failClosed  bool
	stopping    bool
	watchErr    error
	protocolErr error
}

// LoadRequest is one plugin the runtime is asked to bring up.
//
// It carries the three strings the LOAD message needs plus the subscriptions
// the host already validated, and no bundle: the manifest has been read and
// checked long before this point, and handing the whole thing over would give
// a transport a reason to start interpreting it.
type LoadRequest struct {
	// ID is the plugin id from the manifest. The runtime must answer for this
	// exact id.
	ID string

	// BundlePath is the archive on disk. The host has validated it; the runtime
	// is told where it is, never asked to discover it.
	BundlePath string

	// Entry is the manifest's entry point, empty when the manifest declared
	// none — §05 makes the main class optional.
	Entry string

	// Events are the subscriptions the manifest declared. What the runtime
	// reports back is checked against them.
	Events []string
}

// NewSupervisor prepares a supervisor for one runtime process. Nothing is
// started until Start is called, so a runtime can be registered with the host
// long before boot reaches the point where spawning is allowed.
//
// config.Handler is ignored: the supervisor installs its own, because an
// envelope that answers no request is a protocol violation it has to record.
func NewSupervisor(config Config, liveness Liveness) *Supervisor {
	return &Supervisor{config: config, liveness: liveness}
}

// Name is the runtime's short name, as it appears in socket paths and errors.
func (s *Supervisor) Name() string { return s.config.Runtime }

// Version is what the runtime called itself in HELLO. For logs only, never
// parsed. Empty before Start.
func (s *Supervisor) Version() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.child == nil {
		return ""
	}
	return s.child.Version
}

// Start spawns the runtime and completes the handshake, returning only once it
// is ready to be sent work.
//
// ctx bounds startup and startup alone. The heartbeat that follows runs on a
// context of the supervisor's own, because the boot context is cancelled the
// moment boot finishes — watching the runtime on it would stop watching exactly
// when the server starts accepting players.
func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.child != nil {
		s.mu.Unlock()
		return fmt.Errorf("ipc: %s: already started", s.config.Runtime)
	}
	s.mu.Unlock()

	config := s.config
	config.Handler = s.unsolicited

	child, err := Start(ctx, config)
	if err != nil {
		return err
	}

	watchCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.child = child
	s.cancelWatch = cancel
	s.failed = make(chan struct{})
	s.failClosed, s.stopping = false, false
	s.watchErr, s.protocolErr = nil, nil
	s.mu.Unlock()

	go s.watch(watchCtx, child)
	return nil
}

// Load brings up one plugin and returns the events it actually registered.
//
// LOAD is one request and one reply, so callers get ordering for free by
// calling this in sequence — which is what the dependency graph requires and
// what the schema's own comment asks for.
func (s *Supervisor) Load(ctx context.Context, request LoadRequest) ([]string, error) {
	if strings.TrimSpace(request.ID) == "" {
		return nil, fmt.Errorf("ipc: %s: load request has no plugin id", s.config.Runtime)
	}
	conn, err := s.conn()
	if err != nil {
		return nil, err
	}
	reply, err := conn.Request(ctx, &wire.Envelope{Body: &wire.Envelope_Load{Load: &wire.Load{
		PluginId:   request.ID,
		BundlePath: request.BundlePath,
		Entry:      request.Entry,
	}}})
	if err != nil {
		return nil, fmt.Errorf("ipc: %s: load %s: %w", s.config.Runtime, request.ID, err)
	}

	switch body := reply.GetBody().(type) {
	case *wire.Envelope_Loaded:
		if answered := body.Loaded.GetPluginId(); answered != request.ID {
			return nil, fmt.Errorf("ipc: %s: asked to load %s, runtime answered for %s",
				s.config.Runtime, request.ID, answered)
		}
		events := body.Loaded.GetEvents()
		if err := checkDeclared(request, events); err != nil {
			return nil, err
		}
		return events, nil
	case *wire.Envelope_Fail:
		if answered := body.Fail.GetPluginId(); answered != request.ID {
			return nil, fmt.Errorf("ipc: %s: asked to load %s, runtime failed %s",
				s.config.Runtime, request.ID, answered)
		}
		// The reason is written for an admin and is reproduced verbatim: the
		// runtime knows why the plugin would not start and the host does not.
		return nil, fmt.Errorf("plugin %s: %s", request.ID, body.Fail.GetReason())
	default:
		return nil, fmt.Errorf("ipc: %s: %w: answered LOAD with %T",
			s.config.Runtime, ErrProtocol, reply.GetBody())
	}
}

// checkDeclared refuses a plugin that registered an event its manifest never
// declared.
//
// The host dispatches from the manifest, so an undeclared subscription would
// never fire and its author would be left debugging a handler that is simply
// never called. Registering fewer than declared is allowed and silent — a
// plugin may well subscribe conditionally on its own config, and the manifest
// cannot express that.
func checkDeclared(request LoadRequest, registered []string) error {
	if len(registered) == 0 {
		return nil
	}
	declared := make(map[string]struct{}, len(request.Events))
	for _, event := range request.Events {
		declared[event] = struct{}{}
	}
	var undeclared []string
	for _, event := range registered {
		if _, ok := declared[event]; !ok {
			undeclared = append(undeclared, event)
		}
	}
	if len(undeclared) == 0 {
		return nil
	}
	sort.Strings(undeclared)
	return fmt.Errorf("plugin %s: registered %s without declaring it in the manifest",
		request.ID, strings.Join(undeclared, ", "))
}

// Ready ends the load phase. The host opens its listeners only after this, so
// it is sent once every plugin is up and never per plugin.
//
// It is unsolicited by design — the schema has no acknowledgement for it, and
// waiting on one would be inventing protocol.
func (s *Supervisor) Ready() error {
	conn, err := s.conn()
	if err != nil {
		return err
	}
	if err := conn.Send(&wire.Envelope{Body: &wire.Envelope_Ready{Ready: &wire.Ready{}}}); err != nil {
		return fmt.Errorf("ipc: %s: send ready: %w", s.config.Runtime, err)
	}
	return nil
}

// Dispatch delivers one event to one plugin and waits for its verdict.
//
// This is the hot path and the only place a tick thread blocks on another
// process. ctx is what bounds the wait — for a cancellable event that is the
// shared budget of §06, and when it expires the caller decides the outcome from
// the event's own on_failure rather than from anything this returns.
func (s *Supervisor) Dispatch(ctx context.Context, pluginID string, event *abi.Event) (abi.Verdict, error) {
	encoded, err := encodeEvent(event)
	if err != nil {
		return abi.Verdict{}, err
	}
	conn, err := s.conn()
	if err != nil {
		return abi.Verdict{}, err
	}
	reply, err := conn.Request(ctx, &wire.Envelope{Body: &wire.Envelope_Dispatch{Dispatch: &wire.Dispatch{
		PluginId: pluginID,
		Event:    encoded,
	}}})
	if err != nil {
		return abi.Verdict{}, fmt.Errorf("ipc: %s: dispatch %s to %s: %w",
			s.config.Runtime, event.Type, pluginID, err)
	}
	verdict := reply.GetVerdict()
	if verdict == nil {
		return abi.Verdict{}, fmt.Errorf("ipc: %s: %w: answered DISPATCH with %T",
			s.config.Runtime, ErrProtocol, reply.GetBody())
	}
	return decodeVerdict(verdict)
}

// Unload asks the runtime to drop one plugin.
//
// The schema carries no acknowledgement for UNLOAD, so this reports that the
// request was written and nothing more. Whether the plugin actually let go is
// not observable today; Stop is what bounds a runtime that will not.
func (s *Supervisor) Unload(pluginID string) error {
	conn, err := s.conn()
	if err != nil {
		return err
	}
	if err := conn.Send(&wire.Envelope{Body: &wire.Envelope_Unload{Unload: &wire.Unload{
		PluginId: pluginID,
	}}}); err != nil {
		return fmt.Errorf("ipc: %s: unload %s: %w", s.config.Runtime, pluginID, err)
	}
	return nil
}

// Stop asks the runtime to leave and makes sure it did.
//
// The heartbeat is cancelled first. Stopping the other way round races the
// kill inside Child.Stop against a ping that is already in flight, and the
// runtime would be reported unresponsive on its way out of a perfectly orderly
// shutdown.
func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	child, cancel := s.child, s.cancelWatch
	s.stopping = true
	s.child, s.cancelWatch = nil, nil
	s.mu.Unlock()

	if child == nil {
		return nil
	}
	if cancel != nil {
		cancel()
	}
	return child.Stop(ctx)
}

// Failed is closed when the runtime dies on its own — it exited, its connection
// dropped, or it stopped answering pings and was killed. It is not closed by
// Stop: a supervisor that cannot tell a deliberate stop from a dead runtime
// would have its owner react to a shutdown it asked for.
//
// The returned channel is nil before Start, which blocks forever in a select
// and is the correct reading of "this runtime has not failed".
func (s *Supervisor) Failed() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.failed
}

// Err reports why the runtime is gone, or nil while it is still running.
//
// A protocol violation outranks a heartbeat failure: a runtime that sent
// something it should not have was already broken before it stopped answering,
// and reporting the symptom would hide the cause.
func (s *Supervisor) Err() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.protocolErr != nil {
		return s.protocolErr
	}
	return s.watchErr
}

func (s *Supervisor) conn() (*Conn, error) {
	s.mu.RLock()
	child := s.child
	s.mu.RUnlock()
	if child == nil {
		return nil, fmt.Errorf("ipc: %s: runtime is not running", s.config.Runtime)
	}
	return child.Conn(), nil
}

func (s *Supervisor) watch(ctx context.Context, child *Child) {
	err := child.Watch(ctx, s.liveness)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		// Watch lost a race with Stop and reported the process leaving as a
		// failure. It was asked to leave.
		return
	}
	if err != nil && s.watchErr == nil {
		s.watchErr = err
	}
	if s.failed != nil && !s.failClosed {
		s.failClosed = true
		close(s.failed)
	}
}

// unsolicited runs on the read goroutine, so it records and returns rather than
// acting: while it runs, nothing is being read off the socket.
//
// Every envelope the host expects is a reply to a request it made. Anything
// else means the runtime is speaking a protocol this host does not, which does
// not heal on its own — the first one is kept and the rest ignored, because the
// cause is more useful than the count.
func (s *Supervisor) unsolicited(envelope *wire.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.protocolErr == nil {
		s.protocolErr = fmt.Errorf("ipc: %s: %w: sent an unsolicited %T",
			s.config.Runtime, ErrProtocol, envelope.GetBody())
	}
}