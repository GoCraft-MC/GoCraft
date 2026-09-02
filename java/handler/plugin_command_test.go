package handler

import (
	"errors"
	"strings"
	"testing"

	"GoCraft/core/player"
)

// dispatchCapturingReplies runs one line and returns what the sender was told.
func dispatchCapturingReplies(dispatcher *Dispatcher, line string) []string {
	var replies []string
	ctx := CommandContext{
		Player: &player.Player{Username: "oreo"},
		Reply: func(text string) error {
			replies = append(replies, text)
			return nil
		},
	}
	dispatcher.Dispatch(line, ctx)
	return replies
}

func TestDispatchFallsBackToPluginCommands(t *testing.T) {
	dispatcher := NewDispatcher()
	var received string
	dispatcher.SetPluginCommands(func(sender *player.Player, line string) (bool, error) {
		received = line
		return true, nil
	})

	replies := dispatchCapturingReplies(dispatcher, "/shop sell 12.5")
	if received != "shop sell 12.5" {
		t.Fatalf("plugin bridge received %q", received)
	}
	if len(replies) != 0 {
		t.Fatalf("a handled plugin command was answered with %v", replies)
	}
}

// A plugin failure is a sentence for whoever typed the line, shown as it was
// written rather than wrapped in a server-side prefix.
func TestDispatchReportsAPluginCommandFailure(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.SetPluginCommands(func(*player.Player, string) (bool, error) {
		return true, errors.New("price must be between 0.01 and 1000")
	})

	replies := dispatchCapturingReplies(dispatcher, "/shop sell 9000")
	if len(replies) != 1 || !strings.Contains(replies[0], "price must be between") {
		t.Fatalf("plugin failure reported as %v", replies)
	}
}

// A line no plugin owns falls through to the built-in answer, so installing the
// bridge cannot change what an unknown command looks like.
func TestDispatchKeepsUnknownCommandsUnknown(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.SetPluginCommands(func(*player.Player, string) (bool, error) { return false, nil })

	replies := dispatchCapturingReplies(dispatcher, "/nonsense")
	if len(replies) != 1 || !strings.Contains(replies[0], "Unknown command: /nonsense") {
		t.Fatalf("unhandled line reported as %v", replies)
	}
}

// Built-ins win the name, and the bridge is never consulted for one. Plugins
// are refused a built-in name at registration, so this only pins the ordering.
func TestDispatchPrefersABuiltinOverAPlugin(t *testing.T) {
	dispatcher := NewDispatcher()
	ran := false
	dispatcher.Register("ping", func(ctx CommandContext) error {
		ran = true
		return nil
	})
	consulted := false
	dispatcher.SetPluginCommands(func(*player.Player, string) (bool, error) {
		consulted = true
		return true, nil
	})

	dispatchCapturingReplies(dispatcher, "/ping")
	if !ran {
		t.Fatal("the built-in did not run")
	}
	if consulted {
		t.Fatal("the plugin bridge was consulted for a built-in name")
	}
}
