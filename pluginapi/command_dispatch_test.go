package pluginapi

import (
	"errors"
	"testing"

	abi "GoCraft/abi/v1"
)

type wireCommandPlugin struct{}

func (*wireCommandPlugin) OnLoad(context Context) error {
	return context.Commands().Register(7, func(call *CommandContext) error {
		amount, ok := call.Args.Integer("amount")
		if !ok || amount != 3 || call.SenderName != "Console" || call.Sender != nil {
			return errors.New("bad command context")
		}
		call.Reply("created three blocks")
		return errors.New("example failure")
	})
}

func (*wireCommandPlugin) OnEnable() error  { return nil }
func (*wireCommandPlugin) OnDisable() error { return nil }

func TestRuntimeCommandDispatchReturnsRepliesAndErrors(t *testing.T) {
	state := newRuntimeState(Metadata{ID: "commands"}, &wireCommandPlugin{})
	if _, err := state.load("commands", "data"); err != nil {
		t.Fatal(err)
	}
	event := &abi.Event{Type: abi.EventCommandInvoke, Fields: []abi.Value{
		abi.Int64(7), abi.List(), abi.String("Console"),
		abi.List(abi.List(abi.String("amount"), abi.Int64(int64(CommandInteger)), abi.Int64(3))),
	}}
	verdict, err := state.invokeCommand(event)
	if err != nil {
		t.Fatal(err)
	}
	if len(verdict.Effects) != 2 || verdict.Effects[0].Type != abi.HostCallCommandReply ||
		verdict.Effects[0].Fields[0].String != "created three blocks" ||
		verdict.Effects[1].Type != abi.HostCallCommandFailed {
		t.Fatalf("command effects = %#v", verdict.Effects)
	}
}

func TestRuntimeCommandDispatchRejectsMalformedCall(t *testing.T) {
	state := newRuntimeState(Metadata{ID: "commands"}, &wireCommandPlugin{})
	if _, err := state.load("commands", "data"); err != nil {
		t.Fatal(err)
	}
	if _, err := state.invokeCommand(&abi.Event{Type: abi.EventCommandInvoke}); err == nil {
		t.Fatal("malformed command call was accepted")
	}
}
