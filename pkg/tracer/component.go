package tracer

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
	"strings"
)

type Tracer struct {
	tp   trace.TracerProvider
	name string
}

func NewTracer(lc fx.Lifecycle, c *Config) (*Tracer, error) {
	propagators := []propagation.TextMapPropagator{propagation.TraceContext{}}
	if c == nil || !c.DisableBaggagePropagation {
		propagators = append(propagators, propagation.Baggage{})
	}

	propagator := propagation.NewCompositeTextMapPropagator(propagators...)

	if c == nil || !c.Enabled {
		noopTp := noop.NewTracerProvider()
		otel.SetTracerProvider(noopTp)
		otel.SetTextMapPropagator(propagator)

		return &Tracer{tp: noopTp}, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("item-composition-service"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(c.Url),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	spanProc := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithMaxQueueSize(c.BatchSpanProcessor.MaxQueueSize),
		sdktrace.WithBatchTimeout(c.BatchSpanProcessor.BatchTimeout),
		sdktrace.WithExportTimeout(c.BatchSpanProcessor.BatchTimeout),
		sdktrace.WithMaxExportBatchSize(c.BatchSpanProcessor.MaxExportBatchSize),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(spanProc),
	)

	otel.SetTracerProvider(tp)

	tracer := &Tracer{
		tp:   tp,
		name: "item-composition-service",
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		},
	})

	return tracer, nil
}

func (t *Tracer) StartSpan(ctx context.Context, span string) (context.Context, trace.Span) {
	return t.tp.Tracer(t.name).Start(ctx, span)
}

func TraceSafeString(s string) string {
	return strings.ToValidUTF8(s, "?")
}
