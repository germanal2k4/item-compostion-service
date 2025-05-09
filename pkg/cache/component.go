package cache

import (
	"context"
	"fmt"
	"item_compositiom_service/pkg/metrics"
	"sync"
	"time"

	"go.uber.org/zap"
)

type SetGetter[K comparable, V any] interface {
	Set(k K, v V, updateTime time.Time)
	Get(k K) (V, bool)
	LastUpdated(k K) (time.Time, bool)
	CleanUp() int
	Len() int
}

type Cache[K comparable, V any] struct {
	wg     sync.WaitGroup
	mu     sync.Mutex
	closed chan struct{}

	fullUpdateFunc        func(SetGetter[K, V]) error
	incrementalUpdateFunc func(SetGetter[K, V], K) error
	cfg                   config
	setGetter             SetGetter[K, V]
	lgr                   *zap.Logger
	collector             *metricsCollector
}

func New[K comparable, V any](
	lgr *zap.SugaredLogger,
	metrics metrics.MetricsRegistry,
	fullUpdateFunc func(SetGetter[K, V]) error,
	incrementalUpdateFunc func(SetGetter[K, V], K) error,
	opts ...Option,
) *Cache[K, V] {
	cfg := config{
		Name: "default",
		Type: Background,
		TTL:  30 * time.Second,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	label := "background_cache"
	if cfg.Type == LRU {
		label = "lru_cache"
	}

	collector, err := newMetricsCollector(metrics)
	if err != nil {
		lgr.Error("Failed to create metrics", zap.Error(err))
	}

	cache := Cache[K, V]{
		cfg:                   cfg,
		fullUpdateFunc:        fullUpdateFunc,
		incrementalUpdateFunc: incrementalUpdateFunc,
		lgr: lgr.Desugar().With(
			zap.String("component", label),
			zap.String("cache_name", cfg.Name),
		),
		collector: collector,
	}

	if cfg.Type == Background {
		cache.closed = make(chan struct{})
		cache.setGetter = NewBackgroundSetGetter[K, V](cfg.TTL)
	} else {
		cache.setGetter = NewLruSetGetter[K, V](cfg.Capacity, cfg.TTL)
	}

	return &cache
}

func (c *Cache[K, V]) Start(ctx context.Context) (err error) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		c.lgr.Info("Starting first update cache")
		start := time.Now()
		err = c.fullUpdateFunc(c.setGetter)

		dur := time.Since(start)
		c.collector.fullUpdateDurationHistogram.WithLabelValues(c.cfg.Name).Observe(dur.Seconds())

		if err != nil {
			c.collector.cacheErrors.WithLabelValues(c.cfg.Name, "full_update_error").Inc()
			c.lgr.Error("Failed to update cache", zap.Error(err))
			return
		}

		c.collector.cacheSize.WithLabelValues(c.cfg.Name).Set(float64(c.setGetter.Len()))
		c.collector.cacheFullUpdates.WithLabelValues(c.cfg.Name).Inc()
		c.lgr.Info("Cache updated successfully")
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if err == nil {
			c.backgroundUpdate()
		}
		return err
	}
}

func (c *Cache[K, V]) backgroundUpdate() {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		t := time.NewTicker(c.cfg.TTL)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				c.mu.Lock()
				if c.cfg.Type == Background {
					c.lgr.Info("Start full update cache")
					start := time.Now()
					err := c.fullUpdateFunc(c.setGetter)

					dur := time.Since(start)
					c.collector.fullUpdateDurationHistogram.WithLabelValues(c.cfg.Name).Observe(dur.Seconds())

					if err != nil {
						c.collector.cacheErrors.WithLabelValues(c.cfg.Name, "full_update_error").Inc()
						c.lgr.Error("Failed to update cache", zap.Error(err))
					} else {
						c.collector.cacheFullUpdates.WithLabelValues(c.cfg.Name).Inc()
						c.lgr.Info("Cache updated successfully")
					}
				}

				cleaned := c.setGetter.CleanUp()
				if cleaned > 0 {
					c.collector.cacheEvictions.WithLabelValues(c.cfg.Name).Add(float64(cleaned))
					c.lgr.Info(fmt.Sprintf("Cleaned %d objects due ttl", cleaned))
				}

				c.collector.cacheSize.WithLabelValues(c.cfg.Name).Set(float64(c.setGetter.Len()))

				c.mu.Unlock()
			case <-c.closed:
				c.lgr.Info("Background cache closed")
				return
			}
		}
	}()
}

func (c *Cache[K, V]) IncrementalUpdate(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lgr.Info("Start incremental update cache")
	start := time.Now()
	err := c.incrementalUpdateFunc(c.setGetter, key)

	dur := time.Since(start)
	c.collector.incrementalUpdateDurationHistogram.WithLabelValues(c.cfg.Name).Observe(dur.Seconds())

	if err != nil {
		c.collector.cacheErrors.WithLabelValues(c.cfg.Name, "incremental_update_error").Inc()
		c.lgr.Error("Failed to update cache", zap.Error(err))
	} else {
		c.collector.cacheIncrementalUpdates.WithLabelValues(c.cfg.Name).Inc()
		c.lgr.Info("Cache updated successfully")
	}
	c.collector.cacheSize.WithLabelValues(c.cfg.Name).Set(float64(c.setGetter.Len()))
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.setGetter.Get(k)
	if !ok {
		c.collector.cacheMisses.WithLabelValues(c.cfg.Name).Inc()
	} else {
		c.collector.cacheHits.WithLabelValues(c.cfg.Name).Inc()
	}

	return v, ok
}

func (c *Cache[K, V]) Close(ctx context.Context) error {
	close(c.closed)

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
