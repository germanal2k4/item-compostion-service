package metrics

import "github.com/prometheus/client_golang/prometheus"

type NoopMetrics struct{}

type noopRegistry struct{}

func (n *NoopMetrics) GetRegistry() prometheus.Registerer {
	return &noopRegistry{}
}

func (n *noopRegistry) Register(prometheus.Collector) error {
	return nil
}

func (n *noopRegistry) MustRegister(...prometheus.Collector) {
	return
}

func (n *noopRegistry) Unregister(prometheus.Collector) bool {
	return true
}
