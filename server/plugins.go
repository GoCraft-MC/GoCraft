package server

import (
	"context"
	"fmt"
	"log/slog"

	coreplugin "GoCraft/core/plugin"
)

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