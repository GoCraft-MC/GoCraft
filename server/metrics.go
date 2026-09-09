package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	coreworld "GoCraft/core/world"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type serverMetrics struct {
	handler      http.Handler
	tickDuration prometheus.Histogram
}

func newServerMetrics(s *Server) *serverMetrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_players_online",
			Help: "Number of online players across Java and Bedrock editions.",
		}, func() float64 { return float64(s.game.OnlineCount()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_players_max",
			Help: "Configured maximum number of players.",
		}, func() float64 { return float64(s.cfg.MaxPlayers) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_java_connections",
			Help: "Number of active Java TCP connections, including login and status requests.",
		}, func() float64 { return float64(s.connCount.Load()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_tps",
			Help: "Estimated ticks per second from the last 1200 tick processing durations, capped at 20.",
		}, func() float64 {
			tps, _ := s.timings.TPS()
			return tps
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_tick_duration_average_seconds",
			Help: "Average processing duration of the last 1200 ticks in seconds.",
		}, func() float64 {
			_, milliseconds := s.timings.TPS()
			return milliseconds / 1000
		}),
	)
	for dimension, world := range map[string]*coreworld.World{
		"overworld": s.world,
		"nether":    s.netherWorld,
		"end":       s.endWorld,
	} {
		registry.MustRegister(
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name:        "gocraft_chunks_loaded",
				Help:        "Number of chunks currently held in memory by dimension.",
				ConstLabels: prometheus.Labels{"dimension": dimension},
			}, func() float64 { return float64(world.LoadedCount()) }),
			prometheus.NewGaugeFunc(prometheus.GaugeOpts{
				Name:        "gocraft_entities",
				Help:        "Number of non-player entities by dimension.",
				ConstLabels: prometheus.Labels{"dimension": dimension},
			}, func() float64 { return float64(world.Entities.Count()) }),
		)
	}

	tickDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gocraft_tick_duration_seconds",
		Help:    "Processing duration of completed game ticks in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})
	registry.MustRegister(tickDuration)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return &serverMetrics{handler: mux, tickDuration: tickDuration}
}

func (s *Server) startMetricsServer() (*http.Server, error) {
	if !s.cfg.Metrics.Enabled {
		return nil, nil
	}
	listener, err := net.Listen("tcp", s.cfg.Metrics.Address)
	if err != nil {
		return nil, fmt.Errorf("metrics: listening on %s: %w", s.cfg.Metrics.Address, err)
	}
	httpServer := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           s.metrics.handler,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server stopped", "err", err)
		}
	}()
	slog.Info("Prometheus metrics listening", "addr", httpServer.Addr, "path", "/metrics")
	return httpServer, nil
}

func stopMetricsServer(httpServer *http.Server) {
	if httpServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Warn("metrics server shutdown", "err", err)
		_ = httpServer.Close()
	}
}
