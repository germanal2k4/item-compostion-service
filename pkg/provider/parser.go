package provider

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

type GRPCProviderParser struct{}

func NewGRPCProviderParser() *GRPCProviderParser {
	return &GRPCProviderParser{}
}

func (p *GRPCProviderParser) Parse(data []byte) (*ProviderSpec, error) {
	var spec ProviderSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse provider spec: %w", err)
	}

	if err := p.Validate(&spec); err != nil {
		return nil, fmt.Errorf("invalid provider spec: %w", err)
	}

	return &spec, nil
}

func (p *GRPCProviderParser) Validate(spec *ProviderSpec) error {
	if spec.Version == "" {
		return fmt.Errorf("version is required")
	}

	if spec.Kind != "ProviderGRPC" {
		return fmt.Errorf("invalid kind: %s, expected ProviderGRPC", spec.Kind)
	}

	if spec.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if err := p.validateTransport(&spec.Spec.Transport); err != nil {
		return fmt.Errorf("invalid transport config: %w", err)
	}

	if err := p.validatePayload(&spec.Spec.Payload); err != nil {
		return fmt.Errorf("invalid payload config: %w", err)
	}

	if err := p.validateMethods(spec.Spec.Methods); err != nil {
		return fmt.Errorf("invalid methods config: %w", err)
	}

	return nil
}

func (p *GRPCProviderParser) validateTransport(transport *TransportConfig) error {
	if transport.Address == "" {
		return fmt.Errorf("address is required")
	}

	if transport.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

func (p *GRPCProviderParser) validatePayload(payload *PayloadConfig) error {
	if payload.Headers == nil {
		return fmt.Errorf("headers are required")
	}

	for _, value := range payload.Headers {
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
			if os.Getenv(envVar) == "" {
				return fmt.Errorf("environment variable %s is not set", envVar)
			}
		}
	}

	return nil
}

func (p *GRPCProviderParser) validateMethods(methods []MethodConfig) error {
	if len(methods) == 0 {
		return fmt.Errorf("at least one method is required")
	}

	for i, method := range methods {
		if method.Package == "" {
			return fmt.Errorf("method[%d].package is required", i)
		}

		if method.Service == "" {
			return fmt.Errorf("method[%d].service is required", i)
		}

		if method.Method == "" {
			return fmt.Errorf("method[%d].method is required", i)
		}

		if method.Type == "" {
			return fmt.Errorf("method[%d].type is required", i)
		}

		if method.Timeout <= 0 {
			return fmt.Errorf("method[%d].timeout must be positive", i)
		}

		if err := p.validateRequest(&method.Request); err != nil {
			return fmt.Errorf("method[%d].request: %w", i, err)
		}

		if err := p.validateResponse(&method.Response); err != nil {
			return fmt.Errorf("method[%d].response: %w", i, err)
		}
	}

	return nil
}

func (p *GRPCProviderParser) validateRequest(request *RequestConfig) error {
	if request.Domain == "" {
		return fmt.Errorf("domain is required")
	}

	if request.DomainIDs == "" {
		return fmt.Errorf("domain_ids is required")
	}

	return nil
}

func (p *GRPCProviderParser) validateResponse(response *ResponseConfig) error {
	if response.ItemID == "" {
		return fmt.Errorf("itemId is required")
	}

	return nil
}
