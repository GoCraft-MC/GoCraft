package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	corepermission "GoCraft/core/permission"
)

func (e *permissionEditor) apply(reference string) error {
	key, err := extractBytebinKey(reference)
	if err != nil {
		return err
	}
	resp, err := e.client.Get(e.bytebinURL + "/" + key)
	if err != nil {
		return fmt.Errorf("downloading from bytebin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("permission editor session not found or expired")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bytebin returned %s", resp.Status)
	}
	var payload struct {
		Type     string                  `json:"type"`
		Document corepermission.Document `json:"document"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decoding permission data: %w", err)
	}
	if payload.Type != "gocraft-permissions" && payload.Type != "gocraft-permissions-save" {
		return fmt.Errorf("not a GoCraft permission editor link")
	}
	if err := corepermission.Validate(payload.Document); err != nil {
		return fmt.Errorf("invalid permission document: %w", err)
	}
	return e.manager.Replace(payload.Document)
}

func extractBytebinKey(reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if parsed, err := url.Parse(reference); err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		if key := parsed.Query().Get("key"); key != "" {
			return key, nil
		}
		if parsed.Path != "" {
			if base := path.Base(strings.TrimRight(parsed.Path, "/")); base != "" && base != "." {
				return base, nil
			}
		}
	}
	if reference != "" {
		return reference, nil
	}
	return "", fmt.Errorf("invalid permission editor link or key")
}
