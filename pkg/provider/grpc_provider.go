package provider

import (
	"context"
	"fmt"
	"github.com/PaesslerAG/gval"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"time"
)

type GRPCProvider struct {
	spec    *ProviderSpec
	conn    *grpc.ClientConn
	methods map[string]*MethodConfig
}

func NewGRPCProvider(spec *ProviderSpec) (*GRPCProvider, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithNoProxy(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: spec.Spec.Transport.Timeout}
			return dialer.DialContext(ctx, "tcp", addr)
		}),
	}

	conn, err := grpc.NewClient(spec.Spec.Transport.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	methods := make(map[string]*MethodConfig)
	for i := range spec.Spec.Methods {
		method := &spec.Spec.Methods[i]
		methods[method.Method] = method
	}

	return &GRPCProvider{
		spec:    spec,
		conn:    conn,
		methods: methods,
	}, nil
}

func (p *GRPCProvider) GetName() string {
	return p.spec.Metadata.Name
}

func (p *GRPCProvider) GetMethod(methodName string) (*MethodConfig, error) {
	method, exists := p.methods[methodName]
	if !exists {
		return nil, fmt.Errorf("method %s not found", methodName)
	}
	return method, nil
}

func (p *GRPCProvider) ExecuteMethod(ctx context.Context, methodName string, data map[string]interface{}) (interface{}, error) {
	method, err := p.GetMethod(methodName)
	if err != nil {
		return nil, err
	}

	if method.Filter.If != "" {
		matches, err := p.evaluateFilter(method.Filter.If, data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate filter: %w", err)
		}
		if !matches {
			return nil, nil
		}
	}

	_, cancel := context.WithTimeout(ctx, method.Timeout)
	defer cancel()

	// TODO: Implement actual gRPC call based on method configuration

	return nil, fmt.Errorf("not implemented")
}

func (p *GRPCProvider) evaluateFilter(condition string, data map[string]interface{}) (bool, error) {
	expr, err := gval.Evaluate(condition, map[string]interface{}{
		"item": data,
		"time": map[string]interface{}{
			"Now": time.Now,
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	result, ok := expr.(bool)
	if !ok {
		return false, fmt.Errorf("condition did not evaluate to boolean")
	}

	return result, nil
}

func (p *GRPCProvider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
