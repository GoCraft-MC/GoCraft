package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corepermission "GoCraft/core/permission"
	"GoCraft/java/handler"
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

func TestPermissionEditorUploadsAndApplies(t *testing.T) {
	bytebin := newMockBytebin(t)
	manager := corepermission.NewMemory()
	editor := newPermissionEditor(manager, "https://editor.example", bytebin.URL)

	link, err := editor.create([]handler.CommandPermission{{Command: "give", Node: "gocraft.command.give"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(link, "?key=") {
		t.Fatalf("expected link with ?key=, got %q", link)
	}

	// Simulate browser save: download initial data, modify, re-upload
	key := link[strings.Index(link, "?key=")+5:]
	resp, err := http.Get(bytebin.URL + "/" + key)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatal("failed to fetch initial data from mock bytebin")
	}
	var initial bytebinUpload
	_ = json.NewDecoder(resp.Body).Decode(&initial)
	resp.Body.Close()

	initial.Document.Groups["builder"] = corepermission.Group{Permissions: map[string]bool{"gocraft.command.give": true}}
	initial.Document.Users["alex"] = corepermission.User{Groups: []string{"builder"}}

	savePayload, _ := json.Marshal(map[string]interface{}{
		"type":     "gocraft-permissions-save",
		"document": initial.Document,
	})
	saveResp, err := http.Post(bytebin.URL+"/post", "application/json", bytes.NewReader(savePayload))
	if err != nil || saveResp.StatusCode != http.StatusCreated {
		t.Fatal("failed to upload save to mock bytebin")
	}
	var saveResult struct {
		Key string `json:"key"`
	}
	_ = json.NewDecoder(saveResp.Body).Decode(&saveResult)
	saveResp.Body.Close()

	if manager.Allowed("alex", "gocraft.command.give", false, false) {
		t.Fatal("permissions applied before applyedits")
	}
	if err := editor.apply(saveResult.Key); err != nil {
		t.Fatal(err)
	}
	if !manager.Allowed("alex", "gocraft.command.give", false, false) {
		t.Fatal("permissions not applied after applyedits")
	}
}

func TestPermissionEditorRejectsWrongType(t *testing.T) {
	bytebin := newMockBytebin(t)
	manager := corepermission.NewMemory()
	editor := newPermissionEditor(manager, "https://editor.example", bytebin.URL)

	bad, _ := json.Marshal(map[string]string{"type": "something-else", "data": "x"})
	resp, _ := http.Post(bytebin.URL+"/post", "application/json", bytes.NewReader(bad))
	var result struct {
		Key string `json:"key"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if err := editor.apply(result.Key); err == nil {
		t.Fatal("expected error for wrong type, got nil")
	}
}

func TestExtractBytebinKey(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"abc123", "abc123"},
		{"https://editor.example?key=abc123", "abc123"},
		{"https://bytebin.lucko.me/abc123", "abc123"},
	}
	for _, c := range cases {
		got, err := extractBytebinKey(c.input)
		if err != nil || got != c.want {
			t.Errorf("extractBytebinKey(%q) = %q, %v; want %q", c.input, got, err, c.want)
		}
	}
}
