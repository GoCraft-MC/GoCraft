package gocraft

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestCommandsRegisterInvokeAndClear(t *testing.T) {
	var output bytes.Buffer
	commands := newCommands(testLogger(&output), testCommandTree())
	wantErr := errors.New("example failure")
	if err := commands.Register("shop sell <price>", func(call *CommandContext) error {
		call.Reply("hello " + call.Sender.Username)
		return wantErr
	}); err != nil {
		t.Fatal(err)
	}
	if err := commands.Register("shop sell <price>", func(*CommandContext) error { return nil }); err == nil {
		t.Fatal("duplicate path was accepted")
	}
	mistyped := commands.Register("shop sel <price>", func(*CommandContext) error { return nil })
	if mistyped == nil || !strings.Contains(mistyped.Error(), "shop sell <price>") {
		t.Fatalf("a mistyped path was reported as %v", mistyped)
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
	commands := newCommands(testLogger(&output), testCommandTree())
	if err := commands.Register("shop reload", func(*CommandContext) error {
		panic("command exploded")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := commands.invoke(9, &CommandContext{}); err == nil {
		t.Fatal("panicking command returned no error")
	}
	if log := output.String(); !strings.Contains(log, "command exploded") || !strings.Contains(log, "stack") {
		t.Fatalf("panic log = %q", log)
	}
}
