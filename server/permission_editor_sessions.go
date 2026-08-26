package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	corepermission "GoCraft/core/permission"
	"GoCraft/java/handler"
)

type permissionEditSession struct {
	document corepermission.Document
	commands []handler.CommandPermission
	expires  time.Time
	staged   bool
}

type permissionEditor struct {
	mu        sync.Mutex
	manager   *corepermission.Manager
	publicURL string
	lifetime  time.Duration
	sessions  map[string]*permissionEditSession
}

func newPermissionEditor(manager *corepermission.Manager, publicURL string, lifetime time.Duration) *permissionEditor {
	return &permissionEditor{
		manager: manager, publicURL: strings.TrimRight(publicURL, "/"),
		lifetime: lifetime, sessions: make(map[string]*permissionEditSession),
	}
}

func (e *permissionEditor) create(commands []handler.CommandPermission) (string, error) {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	now := time.Now()
	e.mu.Lock()
	for key, session := range e.sessions {
		if now.After(session.expires) {
			delete(e.sessions, key)
		}
	}
	e.sessions[token] = &permissionEditSession{
		document: e.manager.Snapshot(), commands: append([]handler.CommandPermission(nil), commands...),
		expires: now.Add(e.lifetime),
	}
	e.mu.Unlock()
	return e.publicURL + "/permissions/" + token, nil
}

func (e *permissionEditor) snapshot(token string) (permissionEditSession, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	session, ok := e.sessions[token]
	if !ok || time.Now().After(session.expires) {
		delete(e.sessions, token)
		return permissionEditSession{}, false
	}
	copy := *session
	copy.document = corepermission.Clone(session.document)
	copy.commands = append([]handler.CommandPermission(nil), session.commands...)
	return copy, true
}

func (e *permissionEditor) stage(token string, document corepermission.Document) error {
	if err := corepermission.Validate(document); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	session, ok := e.sessions[token]
	if !ok || time.Now().After(session.expires) {
		delete(e.sessions, token)
		return fmt.Errorf("permission edit session expired")
	}
	session.document = corepermission.Clone(document)
	session.staged = true
	return nil
}
