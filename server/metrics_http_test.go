package server

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsRejectConcurrentScrapes(t *testing.T) {
	m := newServerMetrics()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var collect, unblock sync.Once
	m.registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "blocking_probe", Help: "Wait for the concurrent scrape.",
	}, func() float64 {
		collect.Do(func() { close(entered); <-release })
		return 1
	}))
	first := httptest.NewRecorder()
	go func() {
		defer close(done)
		m.handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	}()
	t.Cleanup(func() { unblock.Do(func() { close(release) }); <-done })
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first scrape did not enter the collector")
	}
	second := httptest.NewRecorder()
	m.handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("concurrent scrape status = %d", second.Code)
	}
	unblock.Do(func() { close(release) })
	<-done
	if first.Code != http.StatusOK {
		t.Fatalf("first scrape status = %d", first.Code)
	}
	scrapeMetrics(t, m)
}

func TestMetricsListenerLifecycle(t *testing.T) {
	s := newMetricsTestServer(t)
	if listener, err := s.startMetricsServer(); err != nil || listener != nil {
		t.Fatalf("disabled listener = %v, %v", listener, err)
	}
	s.cfg.Metrics.Enabled = true
	s.cfg.Metrics.Address = "127.0.0.1:0" // Ephemeral test port; config rejects it for operators.
	listener, err := s.startMetricsServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopMetricsServer(listener) })
	client := &http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()
	response, err := client.Get("http://" + listener.Addr + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || !strings.Contains(string(body), "gocraft_players_max 20") {
		t.Fatalf("live scrape = %d, %v: %s", response.StatusCode, err, body)
	}
	stopMetricsServer(listener)
	if conn, err := net.DialTimeout("tcp", listener.Addr, time.Second); err == nil {
		conn.Close()
		t.Fatal("metrics port remained open after shutdown")
	}
	stopMetricsServer(nil)
}
