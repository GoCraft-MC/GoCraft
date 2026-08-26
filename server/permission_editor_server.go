package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

func (s *Server) runPermissionEditor(ctx context.Context) {
	httpServer := &http.Server{
		Addr:              s.cfg.PermissionEditor.Address,
		Handler:           s.permissionEditor,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	shutdownComplete := make(chan struct{})
	go func() {
		defer close(shutdownComplete)
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			slog.Warn("permission editor shutdown failed", "err", err)
		}
	}()
	slog.Info("permission editor listening", "addr", httpServer.Addr,
		"publicURL", s.cfg.PermissionEditor.PublicURL)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("permission editor stopped", "err", err)
	}
	<-shutdownComplete
}
