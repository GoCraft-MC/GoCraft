package server

import (
	"strings"
	"testing"
	"time"

	corepermission "GoCraft/core/permission"
	"GoCraft/core/player"
	"GoCraft/java/handler"
)

func TestGoCraftPermissionEditorCommandRequiresPermission(t *testing.T) {
	dispatcher := handler.NewDispatcher()
	server := &Server{
		cmds:             dispatcher,
		permissionEditor: newPermissionEditor(corepermission.NewMemory(), "https://permissions.example", time.Minute),
	}
	server.registerPermissionCommands()

	operator := player.New([16]byte{1}, "admin", player.ClientEditionBedrock)
	operator.Operator = true
	var reply string
	dispatcher.Dispatch("/gocraft peditor", handler.CommandContext{
		Player: operator,
		Reply:  func(message string) error { reply = message; return nil },
	})
	if !strings.Contains(reply, "https://permissions.example/permissions/") {
		t.Fatalf("operator editor reply = %q", reply)
	}

	reply = ""
	dispatcher.Dispatch("/gocraft peditor", handler.CommandContext{
		Player: player.New([16]byte{2}, "viewer", player.ClientEditionJava),
		Reply:  func(message string) error { reply = message; return nil },
	})
	if !strings.Contains(reply, "permission") {
		t.Fatalf("non-operator editor reply = %q", reply)
	}
}
