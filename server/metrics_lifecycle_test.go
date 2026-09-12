package server

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"GoCraft/config"
	coreworld "GoCraft/core/world"
)

func TestMetricsExistWhenHTTPIsDisabled(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg, err := config.Load("server.yml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.WorldStorage, cfg.WorldSeed, cfg.PreGenerateRadius = config.WorldStorageMemory, 1, 0
	cfg.CustomItems.Enabled = false
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.shutdown)
	if s.metrics == nil || cfg.Metrics.Enabled {
		t.Fatal("disabled HTTP listener prevented collector construction")
	}
	if !strings.Contains(scrapeMetrics(t, s.metrics), "gocraft_plugin_effect_queue_depth 0\n") {
		t.Fatal("plugin subsystem was not registered during construction")
	}
	if listener, err := s.startMetricsServer(); err != nil || listener != nil {
		t.Fatalf("disabled HTTP listener = %v, %v", listener, err)
	}
}

type metricsClosingStorage struct {
	coreworld.Storage
	closed int
}

func (*metricsClosingStorage) Flush() error { return nil }
func (s *metricsClosingStorage) Close() error {
	s.closed++
	return nil
}

func TestStartupFailuresCloseAllWorlds(t *testing.T) {
	for _, failure := range []string{"plugins", "metrics"} {
		t.Run(failure, func(t *testing.T) {
			occupied, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer occupied.Close()
			s := &Server{cfg: &config.Config{}, metrics: newServerMetrics()}
			s.cfg.Metrics = config.MetricsConfig{Enabled: true, Address: occupied.Addr().String()}
			if failure == "plugins" {
				s.cfg.Plugins.Enabled, s.cfg.Plugins.Directory = true, t.TempDir()
				writeBrokenBundle(t, s.cfg.Plugins.Directory)
			}
			var stores []*metricsClosingStorage
			for _, target := range []**coreworld.World{&s.world, &s.netherWorld, &s.endWorld} {
				storage := &metricsClosingStorage{}
				w := coreworld.New(&coreworld.FlatGenerator{}, storage, false)
				*target = w
				stores = append(stores, storage)
				t.Cleanup(func() {
					if storage.closed == 0 {
						_ = w.Close()
					}
				})
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := s.Run(ctx); err == nil || !strings.Contains(err.Error(), failure) {
				t.Fatalf("startup error = %v, want %s failure", err, failure)
			}
			for index, storage := range stores {
				if storage.closed != 1 {
					t.Errorf("world %d was closed %d times", index, storage.closed)
				}
			}
		})
	}
}
