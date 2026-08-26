package server

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"
)

func permissionEditToken(reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if parsed, err := url.Parse(reference); err == nil && parsed.Path != "" {
		reference = path.Base(strings.TrimRight(parsed.Path, "/"))
	}
	decoded, err := base64.RawURLEncoding.DecodeString(reference)
	if err != nil || len(decoded) != 24 {
		return "", fmt.Errorf("invalid permission editor link or code")
	}
	return reference, nil
}

func (e *permissionEditor) apply(reference string) error {
	token, err := permissionEditToken(reference)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	session, ok := e.sessions[token]
	if !ok || time.Now().After(session.expires) {
		delete(e.sessions, token)
		return fmt.Errorf("permission edit session expired")
	}
	if !session.staged {
		return fmt.Errorf("save the permission editor before applying it")
	}
	if !reflect.DeepEqual(e.manager.Snapshot(), session.baseline) {
		return fmt.Errorf("permissions changed since this editor was opened; create a new editor")
	}
	if err := e.manager.Replace(session.document); err != nil {
		return err
	}
	delete(e.sessions, token)
	return nil
}
