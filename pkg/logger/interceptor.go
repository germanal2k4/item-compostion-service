package logger

import (
	"context"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"item_compositiom_service/pkg/recovery"
	"item_compositiom_service/pkg/tracer"
	"strings"
	"sync"
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
	l    *zap.Logger
	opts options
}

func NewInterceptor(l *zap.SugaredLogger) *Interceptor {
	return &Interceptor{
		l: l.Desugar(),
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

		md, mdStr := i.metadataLogField(ctx)
		body, bodyStr := i.messageBodyField(req)

		traceField := zap.Skip()
		spanField := zap.Skip()
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			traceField = zap.String("trace_id", span.SpanContext().TraceID().String())
			spanField = zap.String("span_id", span.SpanContext().SpanID().String())
		}

		lgr := i.l.With(traceField, spanField)

		if !i.opts.disableEnrichTraces {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String(
					"grpc_message_body",
					tracer.TraceSafeString(bodyStr),
				),
				attribute.String(
					"metadata",
					tracer.TraceSafeString(mdStr),
				),
			)
		}

		if !i.opts.disableLogRequest {
			lgr.Info("New incoming request",
				zap.String("component", "server"),
				body,
				md,
			)
		}

		ctx = ToContext(ctx, lgr.Sugar())
		resp, err = handler(ctx, req)

		respBody, respBodyStr := i.messageBodyField(resp)

		if err != nil {
			panicErr := &recovery.PanicError{}

			if errors.As(err, &panicErr) {
				lgr.Error(fmt.Sprintf("Panic occurred: %s", string(panicErr.Stack)),
					zap.String("component", "server"),
					respBody,
					zap.Any("panic_message", panicErr.Panic),
				)
			} else {
				lgr.Error("Got error response",
					zap.String("component", "server"),
					respBody,
					zap.String("error_message", err.Error()),
				)
			}
		}

		if !i.opts.disableLogResponse {
			lgr.Info("Request finished",
				zap.String("component", "server"),
				respBody,
			)
		}

		if !i.opts.disableEnrichTraces && resp != nil {
			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String(
					"grpc_message_response",
					tracer.TraceSafeString(respBodyStr),
				),
			)
		}

		return
	}
}

func (i *Interceptor) messageBodyField(payload any) (zap.Field, string) {
	messageBodyField := zap.Skip()

	p, ok := payload.(proto.Message)
	if !ok {
		i.l.Warn("Payload is not a proto message")
		return messageBodyField, ""
	}

	body, err := protojson.Marshal(p)
	if err != nil {
		i.l.Warn("Error marshalling proto message")
		return messageBodyField, ""
	}

	if len(body) > i.opts.maxMessageSize {
		body = body[:i.opts.maxMessageSize]
		body = append(body, "..."...)
	}

	return zap.String("body", string(body)), string(body)
}

var mdPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

func (i *Interceptor) metadataLogField(ctx context.Context) (zap.Field, string) {
	md, _ := metadata.FromIncomingContext(ctx)
	if md.Len() == 0 {
		return zap.Skip(), ""
	}

	fields := make([]zap.Field, 0, md.Len())
	builder := mdPool.Get().(*strings.Builder)
	for k, v := range md {
		if len(v) == 0 {
			fields = append(fields, zap.String(k, ""))
			builder.WriteString(fmt.Sprintf("%s: ''\n", k))
			continue
		}

		if len(v) == 1 {
			fields = append(fields, zap.String(k, v[0]))
			builder.WriteString(fmt.Sprintf("%s: '%s'\n", k, v[0]))
			continue
		}

		fields = append(fields, zap.Strings(k, v))
		builder.WriteString(fmt.Sprintf("%s: '%s'\n", k, strings.Join(v, ",")))
	}

	return zap.Dict("metadata", fields...), builder.String()
}
