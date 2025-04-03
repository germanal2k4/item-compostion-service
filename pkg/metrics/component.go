package metrics

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsRegistry interface {
	GetRegistry() prometheus.Registerer
}

type Metrics struct {
	wg         sync.WaitGroup
	metricsLgr *metricsErrorLogger
	server     *http.Server

	r *prometheus.Registry
}

func NewMetrics(lc fx.Lifecycle, c *Config, lgr *zap.SugaredLogger) (MetricsRegistry, error) {
	if c == nil || !c.Enable {
		return &NoopMetrics{}, nil
	}

	metrics := &Metrics{
		r: prometheus.NewRegistry(),
		metricsLgr: &metricsErrorLogger{
			lgr.With("component", "metrics"),
		},
	}

	http.Handle("/metrics", promhttp.HandlerFor(metrics.r, promhttp.HandlerOpts{
		ErrorLog: metrics.metricsLgr,
		Registry: metrics.r,
	}))

	metrics.server = &http.Server{
		Addr:    ":" + strconv.Itoa(c.Port),
		Handler: http.DefaultServeMux,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			metrics.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return metrics.Stop(ctx)
		},
	})

	return metrics, nil
}

func (m *Metrics) Start() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.metricsLgr.Errorf("listen http server: %s", err.Error())
		}
	}()
}

func (m *Metrics) Stop(ctx context.Context) error {
	done := make(chan struct{})
	var err error
	go func() {
		err = m.server.Shutdown(context.Background())
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (m *Metrics) GetRegistry() prometheus.Registerer {
	return m.r
}

type metricsErrorLogger struct {
	*zap.SugaredLogger
}

func (m *metricsErrorLogger) Println(v ...interface{}) {
	m.Error(v...)
}
