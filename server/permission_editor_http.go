package server

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	corepermission "GoCraft/core/permission"
	"GoCraft/java/handler"
)

type permissionEditorState struct {
	Document corepermission.Document     `json:"document"`
	Commands []handler.CommandPermission `json:"commands"`
	Expires  string                      `json:"expires"`
}

func (e *permissionEditor) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	secureEditorResponse(response)
	switch request.URL.Path {
	case "/permission-editor/style.css":
		serveEditorAsset(response, "text/css; charset=utf-8", permissionEditorCSS)
		return
	case "/permission-editor/model.js":
		serveEditorAsset(response, "text/javascript; charset=utf-8", permissionEditorModelJS)
		return
	case "/permission-editor/group.js":
		serveEditorAsset(response, "text/javascript; charset=utf-8", permissionEditorGroupJS)
		return
	case "/permission-editor/app.js":
		serveEditorAsset(response, "text/javascript; charset=utf-8", permissionEditorAppJS)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(request.URL.Path, "/permissions/"), "/"), "/")
	if len(parts) < 1 || len(parts) > 2 || parts[0] == "" {
		http.NotFound(response, request)
		return
	}
	token := parts[0]
	session, ok := e.snapshot(token)
	if !ok {
		http.Error(response, "Permission editor link is invalid or expired.", http.StatusGone)
		return
	}
	if len(parts) == 1 {
		if request.Method != http.MethodGet {
			http.Error(response, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}
		serveEditorAsset(response, "text/html; charset=utf-8", permissionEditorHTML)
		return
	}
	if parts[1] != "state" {
		http.NotFound(response, request)
		return
	}
	e.serveState(response, request, token, session)
}

func (e *permissionEditor) serveState(response http.ResponseWriter, request *http.Request, token string, session permissionEditSession) {
	if request.Method == http.MethodGet {
		response.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(response).Encode(permissionEditorState{
			Document: session.document, Commands: session.commands, Expires: session.expires.Format(time.RFC3339),
		})
		return
	}
	if request.Method != http.MethodPut {
		http.Error(response, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	provided := request.Header.Get("X-GoCraft-Editor")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
		http.Error(response, "Missing editor authorization.", http.StatusForbidden)
		return
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var document corepermission.Document
	if err := decoder.Decode(&document); err != nil {
		http.Error(response, "Invalid permission document: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(response, "Only one permission document is allowed.", http.StatusBadRequest)
		return
	}
	if err := e.stage(token, document); err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}
	response.WriteHeader(http.StatusAccepted)
	_, _ = response.Write([]byte("Saved. Apply with: gocraft applyedits " + e.publicURL + "/permissions/" + token))
}
