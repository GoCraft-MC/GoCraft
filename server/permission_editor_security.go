package server

import "net/http"

func secureEditorResponse(response http.ResponseWriter) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Content-Type-Options", "nosniff")
}

func serveEditorAsset(response http.ResponseWriter, contentType string, content []byte) {
	response.Header().Set("Content-Type", contentType)
	_, _ = response.Write(content)
}
