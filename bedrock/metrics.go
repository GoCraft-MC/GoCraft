package bedrock

import "github.com/prometheus/client_golang/prometheus"

// RegisterMetrics registers the adapter's collectors before Listen. A nil
// listener exports zero connections, so disabled Bedrock remains observable.
func (l *Listener) RegisterMetrics(registry prometheus.Registerer) {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "gocraft_bedrock_connections",
		Help: "Number of accepted Bedrock connections after the RakNet and login handshake.",
	}, func() float64 {
		if l == nil {
			return 0
		}
		return float64(l.activeConnections.Load())
	}))
}
