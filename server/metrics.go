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
	registry     *prometheus.Registry
	handler      http.Handler
	tickDuration prometheus.Histogram
}

func newServerMetrics() *serverMetrics {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	tickDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gocraft_tick_duration_seconds",
		Help:    "Processing duration of completed game ticks in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	})
	registry.MustRegister(tickDuration)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		MaxRequestsInFlight: 1,
		Timeout:             5 * time.Second,
	}))
	return &serverMetrics{registry: registry, handler: mux, tickDuration: tickDuration}
}

func (s *Server) registerMetrics() {
	registry := s.metrics.registry
	registry.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_players_max",
			Help: "Configured maximum number of players.",
		}, func() float64 { return float64(s.cfg.MaxPlayers) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_java_connections",
			Help: "Number of active Java TCP connections, including login and status requests.",
		}, func() float64 { return float64(s.connCount.Load()) }),
	)
	s.game.RegisterMetrics(registry)
	for dimension, world := range map[string]*coreworld.World{
		"overworld": s.world,
		"nether":    s.netherWorld,
		"end":       s.endWorld,
	} {
		world.RegisterMetrics(prometheus.WrapRegistererWith(prometheus.Labels{"dimension": dimension}, registry))
	}
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
