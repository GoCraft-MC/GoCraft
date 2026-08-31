package pluginapi

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCommandsRegisterInvokeAndClear(t *testing.T) {
	var output bytes.Buffer
	commands := newCommands(testLogger(&output))
	wantErr := errors.New("example failure")
	if err := commands.Register(7, func(call *CommandContext) error {
		call.Reply("hello " + call.Sender.Username)
		return wantErr
	}); err != nil {
		t.Fatal(err)
	}
	if err := commands.Register(7, func(*CommandContext) error { return nil }); err == nil {
		t.Fatal("duplicate executor was accepted")
	}
	replies, err := commands.invoke(7, &CommandContext{Sender: &Player{Username: "Elias"}})
	if !errors.Is(err, wantErr) || len(replies) != 1 || replies[0] != "hello Elias" {
		t.Fatalf("invoke() = %v, %v", replies, err)
	}
	commands.clear()
	if _, err := commands.invoke(7, &CommandContext{}); err == nil {
		t.Fatal("disabled registry invoked a command")
	}
}

func TestCommandsRecoverPanics(t *testing.T) {
	var output bytes.Buffer
	commands := newCommands(testLogger(&output))
	if err := commands.Register(3, func(*CommandContext) error {
		panic("command exploded")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.invoke(3, &CommandContext{}); err == nil {
		t.Fatal("panicking command returned no error")
	}
	if log := output.String(); !strings.Contains(log, "command exploded") || !strings.Contains(log, "stack") {
		t.Fatalf("panic log = %q", log)
	}
}
