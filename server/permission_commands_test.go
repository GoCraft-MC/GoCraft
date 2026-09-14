package server

import (
	"strings"
	"testing"

	corepermission "GoCraft/core/permission"
	"GoCraft/core/player"
	"GoCraft/java/handler"
	editor "GoCraft/server/permission_editor"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
)

func newMockBytebin(t *testing.T) *httptest.Server {
	t.Helper()
	store := map[string][]byte{}
	counter := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/post":
			counter++
			key := fmt.Sprintf("testkey%d", counter)
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r.Body)
			store[key] = buf.Bytes()
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": key})
		case r.Method == http.MethodGet:
			key := strings.TrimPrefix(r.URL.Path, "/")
			if data, ok := store[key]; ok {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(data)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGoCraftPermissionEditorCommandRequiresPermission(t *testing.T) {
	bytebin := newMockBytebin(t)
	dispatcher := handler.NewDispatcher()
	server := &Server{
		cmds:             dispatcher,
		permissionEditor: editor.NewPermissionEditor(corepermission.NewMemory(), "https://permissions.example", bytebin.URL),
	}
	server.registerPermissionCommands()

	operator := player.New([16]byte{1}, "admin", player.ClientEditionBedrock)
	operator.Operator = true
	var reply string
	dispatcher.Dispatch("/gocraft peditor", handler.CommandContext{
		Player: operator,
		Reply:  func(message string) error { reply = message; return nil },
	})
	if !strings.Contains(reply, "https://permissions.example") || !strings.Contains(reply, "?key=") {
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

func TestGoCraftPermissionEditorUsesLinkReply(t *testing.T) {
	bytebin := newMockBytebin(t)
	dispatcher := handler.NewDispatcher()
	server := &Server{
		cmds: dispatcher,
		permissionEditor: editor.NewPermissionEditor(
			corepermission.NewMemory(), "https://permissions.example", bytebin.URL),
	}
	server.registerPermissionCommands()

	operator := player.New([16]byte{3}, "admin", player.ClientEditionJava)
	operator.Operator = true
	var text, link string
	dispatcher.Dispatch("/gocraft peditor", handler.CommandContext{
		Player: operator,
		ReplyLink: func(message, target string) error {
			text, link = message, target
			return nil
		},
	})
	if text != "Open the permission editor:" || !strings.Contains(link, "?key=") {
		t.Fatalf("link reply = (%q, %q)", text, link)
	}
}
