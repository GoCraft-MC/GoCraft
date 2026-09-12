package game

import (
	"GoCraft/core/player"

	"github.com/prometheus/client_golang/prometheus"
)

// RegisterMetrics registers player collectors before the server starts.
func (g *Game) RegisterMetrics(registry prometheus.Registerer) {
	for label, edition := range map[string]player.ClientEdition{
		"java": player.ClientEditionJava, "bedrock": player.ClientEditionBedrock,
	} {
		registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        "gocraft_players_online",
			Help:        "Number of online players by edition.",
			ConstLabels: prometheus.Labels{"edition": label},
		}, func() float64 {
			count := 0
			g.OnlinePlayers(func(p *player.Player) {
				if p.Edition == edition {
					count++
				}
			})
			return float64(count)
		}))
	}
}
