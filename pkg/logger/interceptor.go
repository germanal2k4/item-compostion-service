package logger

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type options struct {
	disable             bool
	disableEnrichTraces bool
	disableLogRequest   bool
	disableLogResponse  bool
	maxMessageSize      int
}

type LogOption func(o *options)

func WithDisableLogRequest(b bool) LogOption {
	return func(o *options) {
		o.disableLogRequest = b
	}
}

func WithDisableLogResponse(b bool) LogOption {
	return func(o *options) {
		o.disableLogResponse = b
	}
}

func WithDisable(b bool) LogOption {
	return func(o *options) {
		o.disable = b
	}
}

func WithDisableEnrichTraces(b bool) LogOption {
	return func(o *options) {
		o.disableEnrichTraces = b
	}
}

func WithMaxMessageSize(size int) LogOption {
	return func(o *options) {
		o.maxMessageSize = size
	}
}

type Interceptor struct {
	l    *zap.SugaredLogger
	opts options
}

func NewInterceptor(l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		l: l,
	}
}

func (i *Interceptor) GetServerInterceptor(opts ...LogOption) grpc.UnaryServerInterceptor {
	for _, o := range opts {
		o(&i.opts)
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if i.opts.disable {
			return handler(ctx, req)
		}

		traces := zap.Skip()

		if !i.opts.disableEnrichTraces {
			// enrich traces
		}

		if !i.opts.disableLogRequest {
			i.l.Infow("New incoming request",
				"component", "server",
				traces,
				i.messageBodyField(req),
			)
		}

		ctx = ToContext(ctx, i.l)
		resp, err = handler(ctx, req)
		if err != nil && !i.opts.disableLogResponse {
			i.l.Errorw("Got error response",
				"component", "server",
				traces,
				i.messageBodyField(req),
				zap.Error(err),
			)
		}

		if !i.opts.disableLogResponse {
			i.l.Infow("Request finished",
				"component", "server",
				traces,
				i.messageBodyField(resp),
			)
		}

		return
	}
}

func (i *Interceptor) messageBodyField(payload any) zap.Field {
	messageBodyField := zap.Skip()

	p, ok := payload.(proto.Message)
	if !ok {
		i.l.Warnf("Payload is not a proto message")
		return messageBodyField
	}

	body, err := protojson.Marshal(p)
	if err != nil {
		i.l.Warnf("Error marshalling proto message")
		return messageBodyField
	}

	if len(body) > i.opts.maxMessageSize {
		body := body[:i.opts.maxMessageSize]
		body = append(body, "..."...)
	}

	return zap.String("body", string(body))
}
