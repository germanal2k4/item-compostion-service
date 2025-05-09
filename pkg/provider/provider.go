package provider

import (
	"context"
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

type Provider interface {
	GetName() string

	GetMethod(methodName string) (*MethodConfig, error)

	ExecuteMethod(ctx context.Context, methodName string, data map[string]interface{}) (interface{}, error)
}

type Parser interface {
	Parse(data []byte) (*ProviderSpec, error)

	Validate(spec *ProviderSpec) error
}
