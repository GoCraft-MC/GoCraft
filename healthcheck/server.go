package healthcheck

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func handleHealth(state *HealthState) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if state.IsHealthy() {
			writeOk(w)
			return
		}

		writeResponse(w, HealthCheckResponse{
			Status: "unhealthy",
		})
	}
}

func handleLive(state *HealthState) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if state.IsLive() {
			writeOk(w)
			return
		}

		writeResponse(w, HealthCheckResponse{
			Status: "dead",
		})
	}
}

func handleReady(state *HealthState) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if state.IsReady() {
			writeOk(w)
			return
		}

		writeResponse(w, HealthCheckResponse{
			Status: "not ready",
		})
	}
}

const defaultServicePort = "8080"

func getBindAddress() string {
	port := os.Getenv("GOCRAFT_HEALTHCHECK_PORT")
	if port == "" {
		port = defaultServicePort
	}

	addr := port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	return addr
}

func RunHealthCheckServer(ctx context.Context, state *HealthState) func() {
	if enabled := os.Getenv("GOCRAFT_HEALTHCHECK_ENABLED"); strings.EqualFold(enabled, "false") {
		slog.Info("healthcheck server disabled")
		return func() {}
	}

	addr := getBindAddress()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth(state))
	mux.HandleFunc("GET /readyz", handleReady(state))
	mux.HandleFunc("GET /livez", handleLive(state))
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		slog.Info("healthcheck server listening", "address", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("healthcheck server failed to listen", "address", addr, "err", err)
			os.Exit(1)
		}
	}()

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("healthcheck forced shutdown error", "err", err)
		}
	}()

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)

		select {
		case <-shutdownDone:
		case <-time.After(500 * time.Millisecond):
			_ = server.Close()
		}
	}
}
