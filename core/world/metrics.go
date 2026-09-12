package world

import "github.com/prometheus/client_golang/prometheus"

// RegisterMetrics registers world collectors before the server starts. The
// caller supplies the dimension label with prometheus.WrapRegistererWith.
func (w *World) RegisterMetrics(registry prometheus.Registerer) {
	registry.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_chunks_loaded",
			Help: "Number of chunks currently held in memory by dimension.",
		}, func() float64 { return float64(w.LoadedCount()) }),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "gocraft_entities",
			Help: "Number of non-player entities by dimension.",
		}, func() float64 { return float64(w.Entities.Count()) }),
	)
}
