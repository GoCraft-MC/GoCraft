package pluginapi

import (
	"log/slog"

	abi "GoCraft/abi/v1"
	wire "GoCraft/abi/v1/wire"
	"GoCraft/runtime/ipc"
)

type pluginSession struct {
	codec *ipc.Codec
	state *runtimeState
}

func (s *pluginSession) handle(envelope *wire.Envelope) {
	switch body := envelope.GetBody().(type) {
	case *wire.Envelope_Load:
		s.handleLoad(envelope.GetSeq(), body.Load)
	case *wire.Envelope_Dispatch:
		s.handleDispatch(envelope.GetSeq(), body.Dispatch)
	case *wire.Envelope_Invoke:
		s.handleInvoke(envelope.GetSeq(), body.Invoke)
	case *wire.Envelope_Unload:
		if body.Unload.GetPluginId() != s.state.metadata.ID {
			slog.Error("plugin unload id mismatch", "plugin", s.state.metadata.ID,
				"requested", body.Unload.GetPluginId())
			return
		}
		if err := s.state.disable(); err != nil {
			slog.Error("plugin disable failed", "plugin", s.state.metadata.ID, "err", err)
		}
	}
}

func (s *pluginSession) handleLoad(seq uint64, load *wire.Load) {
	events, err := s.state.load(load.GetPluginId(), load.GetDataDirectory())
	if err != nil {
		s.send(&wire.Envelope{Seq: seq, Body: &wire.Envelope_Fail{Fail: &wire.Fail{
			PluginId: load.GetPluginId(), Reason: err.Error(),
		}}})
		return
	}
	s.send(&wire.Envelope{Seq: seq, Body: &wire.Envelope_Loaded{Loaded: &wire.Loaded{
		PluginId: load.GetPluginId(), Events: events,
	}}})
}

func (s *pluginSession) handleDispatch(seq uint64, dispatch *wire.Dispatch) {
	event, err := ipc.DecodeEvent(dispatch.GetEvent())
	verdict := abi.Verdict{}
	if err == nil && dispatch.GetPluginId() != s.state.metadata.ID {
		err = &pluginIDError{expected: s.state.metadata.ID, got: dispatch.GetPluginId()}
	}
	if err == nil {
		verdict, err = s.state.dispatch(event)
	}
	if err != nil {
		if event != nil && event.OnFailure == abi.FailureDeny {
			verdict.Cancelled = true
		}
		slog.Error("plugin event failed", "plugin", s.state.metadata.ID, "err", err)
	}
	encoded, encodeErr := ipc.EncodeVerdict(verdict)
	if encodeErr != nil {
		slog.Error("plugin verdict encoding failed", "plugin", s.state.metadata.ID, "err", encodeErr)
		encoded = &wire.Verdict{Cancelled: verdict.Cancelled}
	}
	s.send(&wire.Envelope{Seq: seq, Body: &wire.Envelope_Verdict{Verdict: encoded}})
}

// handleInvoke answers one command. It always answers: somebody typed a line
// and is waiting on it, so a silence would leave them there until the host gave
// up on a plugin that is working perfectly well.
func (s *pluginSession) handleInvoke(seq uint64, invoke *wire.Invoke) {
	result := abi.CommandResult{}
	invocation, err := ipc.DecodeCommandInvocation(invoke)
	if err == nil && invoke.GetPluginId() != s.state.metadata.ID {
		err = &pluginIDError{expected: s.state.metadata.ID, got: invoke.GetPluginId()}
	}
	if err == nil {
		result = s.state.invokeCommand(invocation)
	} else {
		result.Error = err.Error()
	}
	if result.Error != "" {
		slog.Error("plugin command failed", "plugin", s.state.metadata.ID, "err", result.Error)
	}
	encoded, encodeErr := ipc.EncodeCommandResult(result)
	if encodeErr != nil {
		slog.Error("plugin command encoding failed", "plugin", s.state.metadata.ID, "err", encodeErr)
		encoded = &wire.Invoked{Error: result.Error}
	}
	s.send(&wire.Envelope{Seq: seq, Body: &wire.Envelope_Invoked{Invoked: encoded}})
}

func (s *pluginSession) send(envelope *wire.Envelope) {
	if err := s.codec.Send(envelope); err != nil {
		slog.Error("plugin IPC send failed", "plugin", s.state.metadata.ID, "err", err)
	}
}

type pluginIDError struct{ expected, got string }

func (e *pluginIDError) Error() string {
	return "pluginapi: dispatch id " + e.got + " does not match " + e.expected
}
