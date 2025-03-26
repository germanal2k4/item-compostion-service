package tracer

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type Interceptor struct {
	t *Tracer
}

func NewInterceptor(tracer *Tracer) *Interceptor {
	return &Interceptor{tracer}
}

func (i *Interceptor) GetServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(i.t.name)

	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		fullServiceName := "item_composition.ItemCompositionService"

		ctx, span := tracer.Start(ctx,
			TraceSafeString(fullServiceName+"::"+info.FullMethod),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCServiceKey.String(fullServiceName),
				semconv.RPCMethodKey.String(info.FullMethod),
			),
		)
		defer span.End()

		resp, err := handler(ctx, req)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, TraceSafeString(err.Error()))
		}

		return resp, err
	}
}
