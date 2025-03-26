package tracer

import "time"

type Config struct {
	Enabled                   bool               `yaml:"enabled"`
	DisableBaggagePropagation bool               `yaml:"disable_baggage_propagation"`
	Url                       string             `yaml:"url"`
	BatchSpanProcessor        BatchSpanProcessor `yaml:"batch_span_processor"`
}

type BatchSpanProcessor struct {
	MaxQueueSize       int           `yaml:"max_queue_size"`
	MaxExportBatchSize int           `yaml:"max_export_batch_size"`
	WithBlocking       bool          `yaml:"with_blocking"`
	BatchTimeout       time.Duration `yaml:"batch_timeout"`
	ExportTimeout      time.Duration `yaml:"export_timeout"`
}
