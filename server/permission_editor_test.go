package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	corepermission "GoCraft/core/permission"
	"GoCraft/java/handler"
)

func TestPermissionEditorStagesAndAppliesSingleUseEdits(t *testing.T) {
	manager := corepermission.NewMemory()
	editor := newPermissionEditor(manager, "https://permissions.example", time.Minute)
	link, err := editor.create([]handler.CommandPermission{{Command: "give", Node: "gocraft.command.give"}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(link)
	token := strings.TrimPrefix(parsed.Path, "/permissions/")

	page := httptest.NewRecorder()
	editor.ServeHTTP(page, httptest.NewRequest(http.MethodGet, parsed.Path, nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatalf("editor page response = %d, headers %v", page.Code, page.Header())
	}

	document := manager.Snapshot()
	document.Groups["builder"] = corepermission.Group{Permissions: map[string]bool{"gocraft.command.give": true}}
	document.Users["alex"] = corepermission.User{Groups: []string{"builder"}}
	payload, _ := json.Marshal(document)
	request := httptest.NewRequest(http.MethodPut, parsed.Path+"/state", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-GoCraft-Editor", token)
	saved := httptest.NewRecorder()
	editor.ServeHTTP(saved, request)
	if saved.Code != http.StatusAccepted || !strings.Contains(saved.Body.String(), "applyedits") {
		t.Fatalf("save response = %d %q", saved.Code, saved.Body.String())
	}
	if manager.Allowed("alex", "gocraft.command.give", false, false) {
		t.Fatal("staged edit was applied before confirmation")
	}
	if err := editor.apply(link); err != nil {
		t.Fatal(err)
	}
	if !manager.Allowed("alex", "gocraft.command.give", false, false) {
		t.Fatal("confirmed group permission was not applied")
	}
	if err := editor.apply(link); err == nil {
		t.Fatal("single-use editor link was applied twice")
	}
}

func TestPermissionEditorRejectsUnauthorizedSave(t *testing.T) {
	editor := newPermissionEditor(corepermission.NewMemory(), "https://permissions.example", time.Minute)
	link, err := editor.create(nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(link)
	payload, _ := json.Marshal(editor.manager.Snapshot())
	request := httptest.NewRequest(http.MethodPut, parsed.Path+"/state", bytes.NewReader(payload))
	request.Header.Set("X-GoCraft-Editor", "wrong-token")
	response := httptest.NewRecorder()
	editor.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("unauthorized save status = %d", response.Code)
	}
}
