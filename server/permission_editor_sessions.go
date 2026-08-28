package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	corepermission "GoCraft/core/permission"
	"GoCraft/java/handler"
)

type permissionEditor struct {
	manager    *corepermission.Manager
	editorURL  string
	bytebinURL string
	client     *http.Client
}

func newPermissionEditor(manager *corepermission.Manager, editorURL, bytebinURL string) *permissionEditor {
	return &permissionEditor{
		manager:    manager,
		editorURL:  strings.TrimRight(editorURL, "/"),
		bytebinURL: strings.TrimRight(bytebinURL, "/"),
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

type bytebinUpload struct {
	Type     string                      `json:"type"`
	Document corepermission.Document     `json:"document"`
	Commands []handler.CommandPermission `json:"commands,omitempty"`
}

func (e *permissionEditor) create(commands []handler.CommandPermission) (string, error) {
	payload := bytebinUpload{
		Type:     "gocraft-permissions",
		Document: e.manager.Snapshot(),
		Commands: commands,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := e.client.Post(e.bytebinURL+"/post", "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("uploading to bytebin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bytebin returned %s", resp.Status)
	}
	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding bytebin response: %w", err)
	}
	if result.Key == "" {
		return "", fmt.Errorf("bytebin returned empty key")
	}
	return e.editorURL + "?key=" + result.Key, nil
}
