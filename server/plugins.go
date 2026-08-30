package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	coreplugin "GoCraft/core/plugin"
)

// pluginShutdownTimeout bounds how long unloading may take in total, matching
// the window LoadAll gives its own rollback path.
const pluginShutdownTimeout = 5 * time.Second

// loadPlugins scans the bundle directory, provisions every runtime the scanned
// manifests require, and loads the plugins. It runs before any listener opens:
// a server that accepts players while a spawn-protection plugin is still
// loading is worse than a server that takes longer to boot.
//
// A missing or empty directory is not an error. A failing bundle is: LoadAll
// already rolls back the plugins it started, so refusing to boot leaves the
// admin with a clear failure instead of a silently incomplete server.
func (s *Server) loadPlugins(ctx context.Context) error {
	if !s.cfg.Plugins.Enabled {
		return nil
	}
	bundles, err := coreplugin.ScanBundles(s.cfg.Plugins.Directory)
	if err != nil {
		return fmt.Errorf("scan plugins: %w", err)
	}
	if len(bundles) == 0 {
		slog.Info("plugins: none found", "directory", s.cfg.Plugins.Directory)
		return nil
	}
	if err := s.pluginRegistry.Preflight(ctx, bundles); err != nil {
		return err
	}
	if err := s.pluginRegistry.LoadAll(ctx, bundles); err != nil {
		return err
	}
	slog.Info("plugins: loaded", "count", len(bundles), "directory", s.cfg.Plugins.Directory)
	return nil
}

// unloadPlugins stops every plugin and runtime in reverse load order. It builds
// its own context: Run's is already cancelled by the time shutdown reaches this
// point, and a runtime still needs a bounded window to flush and exit.
//
// Failures are logged rather than returned. The server is going down either
// way, and the world flush that follows matters more than a runtime that
// refused to exit cleanly.
func (s *Server) unloadPlugins() {
	if s.pluginRegistry == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), pluginShutdownTimeout)
	defer cancel()
	if err := s.pluginRegistry.Stop(ctx); err != nil {
		slog.Warn("plugins: unclean shutdown", "err", err)
	}
}
