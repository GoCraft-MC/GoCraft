package server

import _ "embed"

var (
	//go:embed permission_editor_assets/index.html
	permissionEditorHTML []byte
	//go:embed permission_editor_assets/style.css
	permissionEditorCSS []byte
	//go:embed permission_editor_assets/model.js
	permissionEditorModelJS []byte
	//go:embed permission_editor_assets/group.js
	permissionEditorGroupJS []byte
	//go:embed permission_editor_assets/app.js
	permissionEditorAppJS []byte
)
