package setup

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"item_compositiom_service/internal/config"
	"item_compositiom_service/internal/server"
	"item_compositiom_service/internal/services"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/tracer"
)

func Setup(configPath string) (*fx.App, error) {
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return nil, err
	}

	return fx.New(
		fx.StartTimeout(cfg.GrpcConfig.StartDeadline),
		fx.StopTimeout(cfg.GrpcConfig.StopDeadline),
		fx.Provide(
			services.NewService,
			server.NewServer,
			logger.NewLogger,
			logger.NewInterceptor,
			tracer.NewTracer,
			tracer.NewInterceptor,
			metrics.NewMetrics,
			metrics.NewInterceptor,
			func() string {
				return configPath
			},
			func() *server.Config {
				return cfg.GrpcConfig
			},
			func() *logger.Config {
				return cfg.LogConfig
			},
			func() *tracer.Config {
				return cfg.TraceConfig
			},
			func() *metrics.Config {
				return cfg.MetricsConfig
			},
		),
		fx.Invoke(func(*server.Server) {}),
		fx.Invoke(func(l *zap.SugaredLogger) {}),
		fx.Invoke(func(*tracer.Tracer) {}),
		fx.Invoke(func(metrics.MetricsRegistry) {}),
		fx.WithLogger(func(l *zap.SugaredLogger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: l.Desugar()}
		}),
	), nil
}
