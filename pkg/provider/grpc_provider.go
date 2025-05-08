package provider

import (
	"context"
	"fmt"
	"github.com/PaesslerAG/gval"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"net"
	"time"
)

type RetryConfig struct {
	MaxAttempts       int           `json:"maxAttempts"`
	InitialBackoff    time.Duration `json:"initialBackoff"`
	MaxBackoff        time.Duration `json:"maxBackoff"`
	BackoffMultiplier float64       `json:"backoffMultiplier"`
	RetryableCodes    []string      `json:"retryableCodes"`
}

type GRPCProvider struct {
	spec    *ProviderSpec
	conn    *grpc.ClientConn
	methods map[string]*MethodConfig
	retry   *RetryConfig
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

	retryConfig := &RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Second * 10,
		BackoffMultiplier: 2.0,
		RetryableCodes:    []string{"UNAVAILABLE", "DEADLINE_EXCEEDED", "RESOURCE_EXHAUSTED"},
	}

	return &GRPCProvider{
		spec:    spec,
		conn:    conn,
		methods: methods,
		retry:   retryConfig,
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

	ctx, cancel := context.WithTimeout(ctx, method.Timeout)
	defer cancel()

	var result interface{}
	var lastErr error
	backoff := p.retry.InitialBackoff

	for attempt := 1; attempt <= p.retry.MaxAttempts; attempt++ {
		result, lastErr = p.executeGRPCCall(ctx, method, data)
		if lastErr == nil {
			return result, nil
		}

		if st, ok := status.FromError(lastErr); ok {
			isRetryable := false
			for _, code := range p.retry.RetryableCodes {
				if st.Code().String() == code {
					isRetryable = true
					break
				}
			}
			if !isRetryable {
				return nil, lastErr
			}
		}

		if attempt == p.retry.MaxAttempts {
			return nil, fmt.Errorf("failed after %d attempts, last error: %w", attempt, lastErr)
		}

		backoff = time.Duration(float64(backoff) * p.retry.BackoffMultiplier)
		if backoff > p.retry.MaxBackoff {
			backoff = p.retry.MaxBackoff
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}

	return nil, lastErr
}

func (p *GRPCProvider) executeGRPCCall(ctx context.Context, method *MethodConfig, data map[string]interface{}) (interface{}, error) {
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
