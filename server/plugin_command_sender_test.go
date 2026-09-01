package server

import (
	"testing"

	corepermission "GoCraft/core/permission"
	"GoCraft/core/player"
)

func TestPlayerCommandSenderReadsThePermissionManager(t *testing.T) {
	document := corepermission.DefaultDocument()
	document.Users["oreo"] = corepermission.User{Permissions: map[string]bool{"shop.sell": true}}
	manager := corepermission.NewMemory()
	if err := manager.Replace(document); err != nil {
		t.Fatal(err)
	}
	server := &Server{permissions: manager}
	sender := server.commandSenderFor(&player.Player{Username: "oreo", UUID: [16]byte{7}})

	if got := sender.Name(); got != "oreo" {
		t.Fatalf("name = %q", got)
	}
	if got := sender.UUID(); got != ([16]byte{7}) {
		t.Fatalf("uuid = %v", got)
	}
	if !sender.Has("shop.sell") {
		t.Fatal("granted node reported as denied")
	}
	if sender.Has("shop.admin") {
		t.Fatal("ungranted node reported as allowed")
	}
	if _, ok := sender.Player(); !ok {
		t.Fatal("player sender reported no player")
	}
}

// The console has no player and every permission: it is the operator's own
// terminal, and a plugin refusing it a command would be refusing the admin.
func TestConsoleCommandSenderHoldsEveryPermission(t *testing.T) {
	sender := (&Server{}).consoleCommandSender()
	if got := sender.Name(); got != consoleSenderName {
		t.Fatalf("name = %q, want %s", got, consoleSenderName)
	}
	if !sender.Has("anything.at.all") {
		t.Fatal("console was refused a permission")
	}
	if _, ok := sender.Player(); ok {
		t.Fatal("console reported a player")
	}
	if err := sender.SendMessage("hello"); err != nil {
		t.Fatalf("console message = %v", err)
	}
}
