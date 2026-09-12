package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"GoCraft/config"
	corentity "GoCraft/core/entity"
	"GoCraft/core/game"
	"GoCraft/core/player"
	coreworld "GoCraft/core/world"

	"github.com/prometheus/client_golang/prometheus"
)

func newMetricsTestServer(t *testing.T) *Server {
	t.Helper()
	world := func() *coreworld.World {
		w := coreworld.New(&coreworld.FlatGenerator{}, nil, false)
		t.Cleanup(func() { _ = w.Close() })
		return w
	}
	s := &Server{
		cfg: &config.Config{MaxPlayers: 20}, game: game.New(), metrics: newServerMetrics(),
		world: world(), netherWorld: world(), endWorld: world(), timings: newTickTimings(),
	}
	s.registerMetrics()
	s.bedrockListener.RegisterMetrics(s.metrics.registry)
	return s
}

func scrapeMetrics(t *testing.T, m *serverMetrics) string {
	t.Helper()
	response := httptest.NewRecorder()
	m.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("scrape status = %d: %s", response.Code, response.Body.String())
	}
	return response.Body.String()
}

func TestMetricsScrapeReflectsSubsystemState(t *testing.T) {
	s := newMetricsTestServer(t)
	for index, edition := range []player.ClientEdition{player.ClientEditionJava, player.ClientEditionBedrock} {
		if err := s.game.AddPlayer(player.New([16]byte{byte(index + 1)}, "test", edition)); err != nil {
			t.Fatal(err)
		}
	}
	s.world.GetBlock(0, 64, 0)
	s.world.Entities.Add(corentity.New(1, [16]byte{3}, corentity.TypeZombie, 0, 64, 0))
	s.connCount.Store(3)
	s.metrics.observeTick(20*time.Millisecond, [sectionCount]int64{int64(time.Millisecond)})
	s.metrics.registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "subsystem_probe", Help: "Collector registered after the HTTP handler was built.",
	}, func() float64 { return 7 }))
	body := scrapeMetrics(t, s.metrics)
	for _, want := range []string{
		"gocraft_players_online{edition=\"java\"} 1\n", "gocraft_players_online{edition=\"bedrock\"} 1\n",
		"gocraft_players_max 20\n", "gocraft_java_connections 3\n", "gocraft_bedrock_connections 0\n",
		"gocraft_chunks_loaded{dimension=\"overworld\"} 1\n", "gocraft_chunks_loaded{dimension=\"nether\"} 0\n",
		"gocraft_entities{dimension=\"overworld\"} 1\n", "gocraft_entities{dimension=\"end\"} 0\n",
		"gocraft_tick_duration_seconds_count 1\n", "gocraft_tick_duration_seconds_sum 0.02\n",
		"gocraft_tick_section_seconds_sum{section=\"damage\"} 0.001\n", "subsystem_probe 7\n", "go_goroutines ",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape is missing %q", want)
		}
	}
	for _, removed := range []string{"gocraft_tps", "gocraft_tick_duration_average_seconds"} {
		if strings.Contains(body, removed) {
			t.Errorf("scrape still exposes %s", removed)
		}
	}
	s.game.RemovePlayer([16]byte{2})
	if !strings.Contains(scrapeMetrics(t, s.metrics), "gocraft_players_online{edition=\"bedrock\"} 0\n") {
		t.Fatal("departed Bedrock player is still counted")
	}
}

func TestMetricsRecordOneWholeTickDespiteStageFailures(t *testing.T) {
	// Missing subsystems deliberately trigger safeTick's recovery paths.
	s := &Server{metrics: newServerMetrics(), timings: newTickTimings()}
	s.safeTick()
	body := scrapeMetrics(t, s.metrics)
	if !strings.Contains(body, "gocraft_tick_duration_seconds_count 1\n") {
		t.Fatal("a completed tick was skipped or counted more than once")
	}
	if !strings.Contains(body, "gocraft_tick_section_seconds_count{section=\"damage\"} 1\n") {
		t.Fatal("tick sections did not share the completed tick boundary")
	}
}
