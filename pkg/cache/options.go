package cache

import (
	"time"
)

type CacheType int

const (
	Background CacheType = iota
	LRU
)

type config struct {
	Name     string
	Type     CacheType
	TTL      time.Duration
	Capacity int
}

type Option func(*config)

func WithName(name string) Option {
	return func(c *config) {
		c.Name = name
	}
}

func WithLRU() Option {
	return func(c *config) {
		c.Type = LRU
	}
}

func WithCapacity(capacity int) Option {
	return func(c *config) {
		c.Capacity = capacity
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.TTL = ttl
	}
}
