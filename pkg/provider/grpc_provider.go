package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"net"
	"sync"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var ErrrorNoMatch = errors.New("value doesn't match")

type GRPCProvider struct {
	mu      sync.Mutex
	spec    *ProviderSpec
	conn    *grpc.ClientConn
	methods map[string]*MethodConfig
	retry   *RetryConfig

	protoSet   bool
	msgFactory *dynamic.MessageFactory
	stub       grpcdynamic.Stub
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
		methods: methods,
		conn:    conn,
		retry:   retryConfig,
		stub:    grpcdynamic.NewStub(conn),
	}, nil
}

func (p *GRPCProvider) GetName() string {
	return p.spec.Metadata.Name
}

func (p *GRPCProvider) SetProto(proto []byte) error {
	filename := p.spec.Metadata.Name + ".proto"
	parser := protoparse.Parser{
		Accessor: protoparse.FileContentsFromMap(map[string]string{
			filename: string(proto),
		}),
	}

	fds, err := parser.ParseFiles(filename)
	if err != nil {
		return fmt.Errorf("failed to parse proto: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	for _, method := range p.methods {
		service := fds[0].FindService(method.Service)
		if service == nil {
			return fmt.Errorf("service %s not found in proto", method.Service)
		}

		method.desc = service.FindMethodByName(method.Method)
		if method.desc == nil {
			return fmt.Errorf("method %s not found in proto", method.Method)
		}

		file := method.desc.GetFile().AsFileDescriptorProto()
		fd, err := protodesc.NewFile(file, nil)
		if err != nil {
			return fmt.Errorf("failed to create file descriptor for method %s: %w", method.Method, err)
		}

		method.inputMD = fd.Messages().ByName(protoreflect.Name(method.desc.GetInputType().GetName()))
	}
	p.msgFactory = dynamic.NewMessageFactoryWithDefaults()

	p.protoSet = true
	return nil
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
			return nil, ErrrorNoMatch
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
	rd := dynamicpb.NewMessage(method.inputMD)

	if err := mapToDynamic(rd, data); err != nil {
		return nil, fmt.Errorf("заполнение сообщения: %w", err)
	}

	responseMsg, err := p.stub.InvokeRpc(
		ctx,
		method.desc,
		rd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke gRPC method: %w", err)
	}

	jsonResponse, _ := responseMsg.(*dynamic.Message).MarshalJSON()
	var res interface{}
	if err := json.Unmarshal(jsonResponse, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return res, nil
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

func mapToDynamic(msg *dynamicpb.Message, data map[string]interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	u := protojson.UnmarshalOptions{DiscardUnknown: false}
	return u.Unmarshal(raw, msg)
}
