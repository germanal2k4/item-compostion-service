package mongodb

import (
	"context"
	"errors"
	"fmt"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/tracer"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	component            = "mongodb_client"
	componentName string = "MongoDB client"
)

type (
	commandMonitor struct {
		trace        *tracer.Tracer
		traceSpans   sync.Map
		duration     *prometheus.HistogramVec
		count        *prometheus.CounterVec
		failedCount  *prometheus.CounterVec
		runningCount *prometheus.GaugeVec
		config       *LoggingConfig
	}

	eventDTO struct {
		RequestID   int64
		CommandName string
		Duration    *time.Duration
		Failed      bool
		Failure     string
	}
)

func newCommandMonitor(config *MongoStorageConfig, metrics metrics.MetricsRegistry, trace *tracer.Tracer) (*commandMonitor, error) {
	r := metrics.GetRegistry()
	labels := []string{"command_name"}
	m := &commandMonitor{
		trace: trace,
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "mongodb.commands.duration",
			Help: "MongoDB commands duration",
		}, labels),
		count: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mongodb.commands.count",
			Help: "MongoDB commands count",
		}, labels),
		failedCount: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mongodb.commands.failed_count",
			Help: "MongoDB commands failed count",
		}, labels),
		runningCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "mongodb.commands.running_count",
			Help: "MongoDB commands running count",
		}, labels),
		config: config.LoggingConfig,
	}

	err := errors.Join(
		r.Register(m.duration),
		r.Register(m.count),
		r.Register(m.failedCount),
		r.Register(m.runningCount),
	)
	if err != nil {
		return nil, err
	}

	return m, err
}

func (m *commandMonitor) Started(ctx context.Context, event *event.CommandStartedEvent) {
	e := convertEvent(ctx, event)
	m.runningCount.With(map[string]string{"command_name": e.CommandName}).Add(1)

	ctx, span := m.startSpan(ctx, e.CommandName)
	m.traceSpans.Store(e.RequestID, span)
	if m.config.Enabled {
		logRequest := trimLog(event.Command.String(), m.config.QueryMaxBytesToLog)
		ctx = logger.ContextWithKV(ctx, "request", logRequest)
	}

	logger.Infow(ctx, "Command started", "component", component, "command", e.CommandName)
}

func (m *commandMonitor) Succeeded(ctx context.Context, event *event.CommandSucceededEvent) {
	e := convertEvent(ctx, event)
	m.finish(ctx, e)
	logger.Infow(ctx, "Command finished successfully", "component", component, "command", e.CommandName)
}

func (m *commandMonitor) Failed(ctx context.Context, event *event.CommandFailedEvent) {
	e := convertEvent(ctx, event)
	m.finish(ctx, e)
	m.failedCount.With(map[string]string{"command_name": event.CommandName}).Inc()
	logger.Warnw(ctx, "Command failed", "component", component, "command", e.CommandName, "failure", e.Failure)
}

func (m *commandMonitor) finish(_ context.Context, event *eventDTO) {
	if s, ok := m.traceSpans.LoadAndDelete(event.RequestID); ok {
		span := s.(trace.Span)
		if event.Failed {
			span.SetStatus(codes.Error, event.Failure)
		}
		span.End()
	}

	labels := map[string]string{"command_name": event.CommandName}

	m.runningCount.With(labels).Add(-1)
	m.count.With(labels).Inc()
	m.duration.With(labels).Observe(event.Duration.Seconds())
}

func provideCommandMonitor(cm *commandMonitor) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started:   cm.Started,
		Succeeded: cm.Succeeded,
		Failed:    cm.Failed,
	}
}

func convertEvent[rawEvent *event.CommandSucceededEvent | *event.CommandFailedEvent | *event.CommandStartedEvent](ctx context.Context, e rawEvent) *eventDTO {
	ctxCommandName := commandNameFromContext(ctx)
	switch i := any(e).(type) {
	case *event.CommandSucceededEvent:
		if ctxCommandName == "" {
			ctxCommandName = i.CommandName
		}
		return &eventDTO{
			RequestID:   i.RequestID,
			CommandName: ctxCommandName,
			Duration:    &[]time.Duration{time.Duration(i.Duration) * time.Nanosecond}[0],
		}
	case *event.CommandFailedEvent:
		if ctxCommandName == "" {
			ctxCommandName = i.CommandName
		}
		return &eventDTO{
			RequestID:   i.RequestID,
			CommandName: ctxCommandName,
			Duration:    &[]time.Duration{time.Duration(i.Duration) * time.Nanosecond}[0],
			Failed:      true,
			Failure:     i.Failure,
		}
	case *event.CommandStartedEvent:
		if ctxCommandName == "" {
			ctxCommandName = i.CommandName
		}
		return &eventDTO{
			RequestID:   i.RequestID,
			CommandName: ctxCommandName,
			Duration:    nil,
		}
	}
	return nil
}

func (m *commandMonitor) startSpan(ctx context.Context, commandName string) (context.Context, trace.Span) {
	ctx, span := m.trace.StartSpan(ctx, componentName)
	kv := attribute.String(fmt.Sprintf("%s.%s", component, "command_name"), commandName)
	span.SetAttributes(kv)
	return ctx, span
}

func trimLog(s string, logLen int) string {
	logString := []rune(s)
	if len(logString) > logLen {
		logString = logString[:logLen]
	}
	return string(logString)
}

type mongoCommandNameKeyType struct{}

var mongoCommandNameKey = mongoCommandNameKeyType{}

func WithCommandName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, mongoCommandNameKey, name)
}

func commandNameFromContext(ctx context.Context) string {
	commandName := ctx.Value(mongoCommandNameKey)
	switch x := commandName.(type) {
	case string:
		return x
	default:
		return ""
	}
}
