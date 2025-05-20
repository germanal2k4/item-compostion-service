package provider

import (
	"github.com/jhump/protoreflect/desc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"time"
)

type TransportConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
	Logging struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"logging"`
}

type PayloadConfig struct {
	Headers map[string]string `yaml:"headers"`
}

type MethodConfig struct {
	Package  string                         `yaml:"package"`
	Service  string                         `yaml:"service"`
	Method   string                         `yaml:"method"`
	Type     ProviderType                   `yaml:"type"`
	Timeout  time.Duration                  `yaml:"timeout"`
	Filter   FilterConfig                   `yaml:"filter"`
	Request  map[string]string              `yaml:"request"`
	Response map[string]string              `yaml:"response"`
	desc     *desc.MethodDescriptor         `yaml:"-"`
	inputMD  protoreflect.MessageDescriptor `yaml:"-"`
}

type FilterConfig struct {
	If string `yaml:"if"`
}
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
