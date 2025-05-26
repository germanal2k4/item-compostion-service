package provider

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
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

	if spec.Spec.Payload.Headers != nil {
		for k, v := range spec.Spec.Payload.Headers {
			if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
				envVar := strings.TrimSuffix(strings.TrimPrefix(v, "${"), "}")
				if value := os.Getenv(envVar); value != "" {
					spec.Spec.Payload.Headers[k] = value
				}
			}
		}
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

	for key, value := range payload.Headers {
		if key == "" {
			return fmt.Errorf("header key cannot be empty")
		}

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

		if err := p.validateFilter(&method.Filter); err != nil {
			return fmt.Errorf("method[%d].filter: %w", i, err)
		}

		if err := p.validateRequestResponse(method.Request, "request", i); err != nil {
			return err
		}

		if err := p.validateRequestResponse(method.Response, "response", i); err != nil {
			return err
		}
	}

	return nil
}

func (p *GRPCProviderParser) validateFilter(filter *FilterConfig) error {
	if filter.If == "" {
		return nil
	}

	validExpr := regexp.MustCompile(`^[a-zA-Z0-9\s\.\(\)\+\-\*\/\>\<\=\!\&\|\:\"\'\,\[\]\{\}]+$`)
	if !validExpr.MatchString(filter.If) {
		return fmt.Errorf("invalid filter expression: %s", filter.If)
	}

	return nil
}

func (p *GRPCProviderParser) validateRequestResponse(mapping map[string]string, kind string, methodIndex int) error {
	if mapping == nil {
		return fmt.Errorf("method[%d].%s is required", methodIndex, kind)
	}

	for field, path := range mapping {
		if field == "" {
			return fmt.Errorf("method[%d].%s: field name cannot be empty", methodIndex, kind)
		}

		if path == "" {
			return fmt.Errorf("method[%d].%s: path for field %s cannot be empty", methodIndex, kind, field)
		}

		validPath := regexp.MustCompile(`^[a-zA-Z0-9\.\[\]\"\']+$`)
		if !validPath.MatchString(path) {
			return fmt.Errorf("method[%d].%s: invalid path expression for field %s: %s", methodIndex, kind, field, path)
		}
	}

	return nil
}
