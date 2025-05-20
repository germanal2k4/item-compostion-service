package parser

import (
	"item_compositiom_service/pkg/metrics"

	"errors"
	"github.com/prometheus/client_golang/prometheus"
)

type metricsCollector struct {
	parseTime          prometheus.HistogramVec
	adjustTime         prometheus.HistogramVec
	errorsCount        prometheus.CounterVec
	parseRequestCount  prometheus.CounterVec
	adjustRequestCount prometheus.CounterVec
}

func newMetricsCollector(registry metrics.MetricsRegistry) (*metricsCollector, error) {
	metrics := &metricsCollector{}

	metrics.parseTime = *prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "parser_parse_time_seconds",
		Help: "Time taken to parse templates",
	}, []string{})

	metrics.adjustTime = *prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "parser_adjust_time_seconds",
		Help: "Time taken to adjust templates",
	}, []string{})

	metrics.errorsCount = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "parser_errors_total",
		Help: "Total number of parser errors",
	}, []string{"error_type"})

	metrics.parseRequestCount = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "parser_parse_requests_total",
		Help: "Total number of parse requests",
	}, []string{})

	metrics.adjustRequestCount = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "parser_adjust_requests_total",
		Help: "Total number of adjust requests",
	}, []string{})

	r := registry.GetRegistry()

	err := errors.Join(
		r.Register(metrics.parseTime),
		r.Register(metrics.adjustTime),
		r.Register(metrics.errorsCount),
		r.Register(metrics.parseRequestCount),
		r.Register(metrics.adjustRequestCount),
	)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}
