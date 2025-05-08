package provider

import (
	"context"
	"time"
)

type ProviderType string

const (
	TypeBatch       ProviderType = "Batch"
	TypeDomainBatch ProviderType = "DomainBatch"
	TypeItem        ProviderType = "Item"
)

type ProviderSpec struct {
	Version  string           `yaml:"version"`
	Kind     string           `yaml:"kind"`
	Metadata ProviderMetadata `yaml:"metadata"`
	Spec     ProviderConfig   `yaml:"spec"`
}

type ProviderMetadata struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
}

type ProviderConfig struct {
	Transport TransportConfig `yaml:"transport"`
	Payload   PayloadConfig   `yaml:"payload"`
	Methods   []MethodConfig  `yaml:"methods"`
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
	Package  string         `yaml:"package"`
	Service  string         `yaml:"service"`
	Method   string         `yaml:"method"`
	Type     ProviderType   `yaml:"type"`
	Timeout  time.Duration  `yaml:"timeout"`
	Filter   FilterConfig   `yaml:"filter"`
	Request  RequestConfig  `yaml:"request"`
	Response ResponseConfig `yaml:"response"`
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

type Provider interface {
	GetName() string

	GetMethod(methodName string) (*MethodConfig, error)

	ExecuteMethod(ctx context.Context, methodName string, data map[string]interface{}) (interface{}, error)
}

type Parser interface {
	Parse(data []byte) (*ProviderSpec, error)

	Validate(spec *ProviderSpec) error
}
