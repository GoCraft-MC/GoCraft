package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"GoCraft/config"
	coreplugin "GoCraft/core/plugin"
	"GoCraft/runtime/jvm"
)

// pluginShutdownTimeout bounds how long unloading may take in total, matching
// the window LoadAll gives its own rollback path.
const pluginShutdownTimeout = 5 * time.Second

// pluginTickRate is the simulation rate published to every runtime in WELCOME,
// matching the 50ms ticker in Run. A runtime cannot enforce anything with it;
// it is what lets a plugin convert ticks to seconds without guessing.
const pluginTickRate = 20

// registerPluginRuntimes makes the language backends available to the registry.
//
// Registering one costs nothing. It is constructed, not started: no process is
// spawned, no JDK is looked for and nothing is downloaded until Preflight finds
// a scanned manifest that asks for it by name. That is what keeps "a server
// with no Java plugin never touches Java" true rather than aspirational, and it
// is why this runs unconditionally rather than behind a config switch.
func registerPluginRuntimes(registry *coreplugin.Registry, cfg *config.Config) error {
	java := cfg.Plugins.Runtimes.JVM
	return registry.RegisterRuntime(jvm.New(jvm.Config{
		JavaPath:     java.JavaPath,
		PreferSystem: java.PreferSystem,
		JarPath:      java.JarPath,
		TickRate:     pluginTickRate,
		EventBudget:  time.Duration(cfg.Plugins.EventBudgetMillis) * time.Millisecond,
	}))
}

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
