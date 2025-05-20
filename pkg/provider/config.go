package provider

import (
	"time"

	"github.com/jhump/protoreflect/desc"
)

type ProviderConfig struct {
	Transport TransportConfig `yaml:"transport"`
	Payload   PayloadConfig   `yaml:"payload"`
	Methods   []MethodConfig  `yaml:"methods"`
}

type RetryConfig struct {
	MaxAttempts       int           `json:"maxAttempts"`
	InitialBackoff    time.Duration `json:"initialBackoff"`
	MaxBackoff        time.Duration `json:"maxBackoff"`
	BackoffMultiplier float64       `json:"backoffMultiplier"`
	RetryableCodes    []string      `json:"retryableCodes"`
}

type TransportConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
	Logging LoggingConfig `yaml:"logging"`
}

type LoggingConfig struct {
	Enabled bool `yaml:"enabled"`
}

type PayloadConfig struct {
	Headers map[string]string `yaml:"headers"`
}

type MethodConfig struct {
	Package  string                 `yaml:"package"`
	Service  string                 `yaml:"service"`
	Method   string                 `yaml:"method"`
	Type     ProviderType           `yaml:"type"`
	Timeout  time.Duration          `yaml:"timeout"`
	Filter   FilterConfig           `yaml:"filter"`
	Request  RequestConfig          `yaml:"request"`
	Response ResponseConfig         `yaml:"response"`
	desc     *desc.MethodDescriptor `yaml:"-"`
}

type FilterConfig struct {
	If string `yaml:"if"`
}

type RequestConfig struct {
	Domain    string `yaml:"domain"`
	DomainIDs string `yaml:"domain_ids"`
}

type ResponseConfig struct {
	ItemID string `yaml:"itemId"`
}
