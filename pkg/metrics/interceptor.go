package metrics

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type Interceptor struct {
	totalRequests   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorsTotal     *prometheus.CounterVec
}

func NewInterceptor(m MetricsRegistry) (*Interceptor, error) {
	namespace := "grpc_server"

	i := &Interceptor{
		totalRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "The total number of grpc requests",
			},
			[]string{"method", "code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration",
				Help:      "The grpc request duration",
			},
			[]string{"method"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "The total number of errors",
			},
			[]string{"method", "code"},
		),
	}

	registry := m.GetRegistry()

	err := errors.Join(
		registry.Register(i.totalRequests),
		registry.Register(i.requestDuration),
		registry.Register(i.errorsTotal),
	)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (i *Interceptor) GetServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (interface{}, error) {

		startTime := time.Now()
		method := info.FullMethod

		resp, err := handler(ctx, req)

		dur := time.Since(startTime)
		statusCode := status.Code(err)

		i.requestDuration.WithLabelValues(method).Observe(dur.Seconds())
		i.totalRequests.WithLabelValues(method, statusCode.String()).Inc()

		if statusCode != codes.OK {
			i.errorsTotal.WithLabelValues(method, statusCode.String()).Inc()
		}

		return resp, err
	}
}
