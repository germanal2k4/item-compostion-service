package logger

import "time"

type Config struct {
	LogLevel      string         `yaml:"log_level"`
	FilePath      string         `yaml:"file_path"`
	Transport     string         `yaml:"transport"`
	EncodeTime    string         `yaml:"encode_time"`
	DevMode       bool           `yaml:"dev_mode"`
	ElasticConfig *ElasticConfig `yaml:"elastic_config"`
}

type ElasticConfig struct {
	Url             string        `yaml:"url"`
	Index           string        `yaml:"index"`
	WriteBufferSize int           `yaml:"write_buffer_size"`
	FlushInterval   time.Duration `yaml:"flush_interval"`
}
