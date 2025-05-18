package localdb

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

type metricsCollector struct {
	readCount    prometheus.CounterVec
	readDuration prometheus.HistogramVec
	errorsCount  prometheus.CounterVec
	updateCount  prometheus.CounterVec
}

func newMetricsCollector(r prometheus.Registerer) (*metricsCollector, error) {
	m := &metricsCollector{
		readCount: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "read_count",
			Help: "Number of reads files from the localdb",
		}, []string{"collection"}),
		readDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "read_duration_seconds",
			Help: "Duration of reads files from the localdb",
		}, []string{"collection"}),
		errorsCount: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "errors_count",
			Help: "Number of errors from the localdb",
		}, []string{"collection", "type"}),
		updateCount: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "update_count",
			Help: "Number of updates requested to the localdb",
		}, []string{"collection", "type"}),
	}

	err := errors.Join(
		r.Register(m.readCount),
		r.Register(m.readDuration),
		r.Register(m.errorsCount),
		r.Register(m.updateCount),
	)

	return m, err
}
