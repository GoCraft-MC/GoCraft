package customitems

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// JavaPackServer serves the generated Java resource pack over HTTP so that Java
// clients can download it. The URL and SHA-1 hash are then passed to the Java
// resource_pack_push packet.
type JavaPackServer struct {
	data []byte
	hash [20]byte
	srv  *http.Server
	url  string
}

// StartJavaPackServer starts an HTTP server on the given port and returns a
// JavaPackServer whose URL and Hash fields are ready to use.
//
// publicHost is the hostname or IP that Java clients will use to reach the
// server (e.g. "203.0.113.5"). If empty, the loopback address is used which
// only works for local testing.
func StartJavaPackServer(data []byte, hash [20]byte, publicHost string, port int) (*JavaPackServer, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no pack data")
	}

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}

	if publicHost == "" {
		publicHost = "127.0.0.1"
	}

	hashHex := hex.EncodeToString(hash[:])
	packURL := fmt.Sprintf("http://%s:%d/%s.zip", publicHost, port, hashHex)

	mux := http.NewServeMux()
	mux.HandleFunc("/"+hashHex+".zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		_, _ = w.Write(data)
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Warn("customitems: java pack HTTP server error", "err", err)
		}
	}()

	slog.Info("customitems: Java pack server started", "url", packURL)
	return &JavaPackServer{data: data, hash: hash, srv: srv, url: packURL}, nil
}

// URL returns the full HTTP URL clients use to download the pack.
func (s *JavaPackServer) URL() string { return s.url }

// HashHex returns the SHA-1 hash of the pack as a lowercase hex string.
func (s *JavaPackServer) HashHex() string { return hex.EncodeToString(s.hash[:]) }

// Stop shuts down the HTTP server gracefully.
func (s *JavaPackServer) Stop(ctx context.Context) { _ = s.srv.Shutdown(ctx) }
