package cache

import (
	"errors"
	"item_compositiom_service/pkg/metrics"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	collector     *metricsCollector
	initCollector sync.Once
)

type metricsCollector struct {
	cacheSize                          prometheus.GaugeVec
	cacheHits                          prometheus.CounterVec
	cacheMisses                        prometheus.CounterVec
	cacheEvictions                     prometheus.CounterVec
	cacheErrors                        prometheus.CounterVec
	cacheFullUpdates                   prometheus.CounterVec
	cacheIncrementalUpdates            prometheus.CounterVec
	fullUpdateDurationHistogram        prometheus.HistogramVec
	incrementalUpdateDurationHistogram prometheus.HistogramVec
}

func newMetricsCollector(registry metrics.MetricsRegistry) (*metricsCollector, error) {
	metrics := &metricsCollector{}

	metrics.cacheSize = *prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cache_size",
		Help: "The size of the cache",
	}, []string{"cache_name"})

	metrics.cacheHits = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_hits",
		Help: "The number of cache hits",
	}, []string{"cache_name"})

	metrics.cacheMisses = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_misses",
		Help: "The number of cache misses",
	}, []string{"cache_name"})

	metrics.cacheEvictions = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_evictions",
		Help: "The number of cache evictions",
	}, []string{"cache_name"})

	metrics.cacheErrors = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_errors",
		Help: "The number of cache errors",
	}, []string{"cache_name", "error_type"})

	metrics.cacheFullUpdates = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_full_updates",
		Help: "The number of cache full updates",
	}, []string{"cache_name"})

	metrics.cacheIncrementalUpdates = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_incremental_updates",
		Help: "The number of cache incremental updates",
	}, []string{"cache_name"})

	metrics.fullUpdateDurationHistogram = *prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cache_full_update_duration_seconds",
		Help: "The duration of cache full updates",
	}, []string{"cache_name"})

	metrics.incrementalUpdateDurationHistogram = *prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cache_incremental_update_duration_seconds",
		Help: "The duration of cache incremental updates",
	}, []string{"cache_name"})

	r := registry.GetRegistry()

	err := errors.Join(r.Register(metrics.cacheSize),
		r.Register(metrics.cacheHits),
		r.Register(metrics.cacheMisses),
		r.Register(metrics.cacheEvictions),
		r.Register(metrics.cacheErrors),
		r.Register(metrics.cacheFullUpdates),
		r.Register(metrics.cacheIncrementalUpdates),
		r.Register(metrics.fullUpdateDurationHistogram),
		r.Register(metrics.incrementalUpdateDurationHistogram),
	)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}
