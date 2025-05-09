package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type SetGetter[K comparable, V any] interface {
	Set(k K, v V, updateTime time.Time)
	Get(k K) (V, bool)
	LastUpdated(k K) (time.Time, bool)
	CleanUp() int
}

type Cache[K comparable, V any] struct {
	wg     sync.WaitGroup
	closed chan struct{}

	fullUpdateFunc        func(SetGetter[K, V]) error
	incrementalUpdateFunc func(SetGetter[K, V], K) error
	lgr                   *zap.Logger
	cfg                   config
	setGetter             SetGetter[K, V]
}

func New[K comparable, V any](
	lgr *zap.SugaredLogger,
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

	cache := Cache[K, V]{
		cfg: cfg,
		lgr: lgr.Desugar().With(
			zap.String("component", label),
			zap.String("cache_name", cfg.Name),
		),
		fullUpdateFunc:        fullUpdateFunc,
		incrementalUpdateFunc: incrementalUpdateFunc,
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
		if err = c.fullUpdateFunc(c.setGetter); err != nil {
			c.lgr.Error("Failed to update cache", zap.Error(err))
			return
		}
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
				if c.cfg.Type == Background {
					c.lgr.Info("Start full update cache")
					err := c.fullUpdateFunc(c.setGetter)
					if err != nil {
						c.lgr.Error("Failed to update cache", zap.Error(err))
					} else {
						c.lgr.Info("Cache updated successfully")
					}
				}

				cleaned := c.setGetter.CleanUp()
				if cleaned > 0 {
					c.lgr.Info(fmt.Sprintf("Cleaned %d objects due ttl", cleaned))
				}
			case <-c.closed:
				c.lgr.Info("Background cache closed")
				return
			}
		}
	}()
}

func (c *Cache[K, V]) IncrementalUpdate(key K) {
	c.lgr.Info("Start incremental update cache")
	err := c.incrementalUpdateFunc(c.setGetter, key)
	if err != nil {
		c.lgr.Error("Failed to update cache", zap.Error(err))
	} else {
		c.lgr.Info("Cache updated successfully")
	}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.setGetter.Get(k)
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
