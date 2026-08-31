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
		if !call.Can("example.build") {
			return errors.New("the resolved permission did not arrive")
		}
		call.Reply("created three blocks")
		return errors.New("example failure")
	})
}

func (*wireCommandPlugin) OnEnable() error  { return nil }
func (*wireCommandPlugin) OnDisable() error { return nil }

func consoleInvocation() abi.CommandInvocation {
	return abi.CommandInvocation{
		Executor: 7,
		Sender: abi.CommandSender{
			Player: abi.List(),
			Name:   "Console",
			Permissions: []abi.Value{
				abi.List(abi.String("example.build"), abi.Bool(true)),
			},
		},
		Arguments: []abi.CommandArgument{
			{Name: "amount", Type: abi.CommandArgumentInteger, Value: abi.Int64(3)},
		},
	}
}

// A handler that replied and then failed has to produce both: the reply is a
// sentence it already wrote for the sender, and dropping it because the command
// also failed would leave them with nothing to read.
func TestRuntimeCommandDispatchReturnsRepliesAndErrors(t *testing.T) {
	state := newRuntimeState(Metadata{ID: "commands"}, &wireCommandPlugin{})
	if _, err := state.load("commands", "data"); err != nil {
		t.Fatal(err)
	}

	result := state.invokeCommand(consoleInvocation())

	if result.Error != "example failure" {
		t.Fatalf("invokeCommand() error = %q", result.Error)
	}
	// A reply is a chat.message the host queues, the same host call an event
	// handler produces. There is no command-shaped reply message any more.
	if len(result.Effects) != 1 || result.Effects[0].Type != hostCallMessage ||
		result.Effects[0].Fields[1].String != "created three blocks" {
		t.Fatalf("command effects = %#v", result.Effects)
	}
}

// An argument with no type never reaches a handler. It would be read as
// whichever field the plugin asked for, which is a value rather than a failure.
func TestRuntimeCommandDispatchRejectsMalformedCall(t *testing.T) {
	state := newRuntimeState(Metadata{ID: "commands"}, &wireCommandPlugin{})
	if _, err := state.load("commands", "data"); err != nil {
		t.Fatal(err)
	}

	invocation := consoleInvocation()
	invocation.Arguments[0].Type = abi.CommandArgumentInvalid
	if result := state.invokeCommand(invocation); result.Error == "" {
		t.Fatal("malformed command call was accepted")
	}
}

// An executor the plugin never registered is answered, not ignored. Whoever
// typed the command is waiting on the reply either way.
func TestRuntimeCommandDispatchRejectsAnUnregisteredExecutor(t *testing.T) {
	state := newRuntimeState(Metadata{ID: "commands"}, &wireCommandPlugin{})
	if _, err := state.load("commands", "data"); err != nil {
		t.Fatal(err)
	}

	invocation := consoleInvocation()
	invocation.Executor = 99
	if result := state.invokeCommand(invocation); result.Error == "" {
		t.Fatal("an unregistered executor was accepted")
	}
}
