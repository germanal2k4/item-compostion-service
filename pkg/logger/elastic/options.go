package elastic

import "time"

const bulkSize = 100

type config struct {
	Url             string
	Index           string
	WriteBufferSize int
	FlushInterval   time.Duration
}

type Option func(*config)

func WithUrl(url string) Option {
	return func(c *config) {
		c.Url = url
	}
}

func WithIndex(index string) Option {
	return func(c *config) {
		c.Index = index
	}
}

func WithWriteBufferSize(writeBufferSize int) Option {
	return func(c *config) {
		c.WriteBufferSize = writeBufferSize
	}
}

func WithFlushInterval(flushInterval time.Duration) Option {
	return func(c *config) {
		c.FlushInterval = flushInterval
	}
}
