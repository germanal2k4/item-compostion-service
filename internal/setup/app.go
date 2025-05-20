package setup

import (
	"item_compositiom_service/internal/config"
	"item_compositiom_service/internal/repository"
	localdb "item_compositiom_service/internal/repository/local_db"
	mongodb "item_compositiom_service/internal/repository/mongo_db"
	"item_compositiom_service/internal/server"
	"item_compositiom_service/internal/services"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/tracer"

	"go.uber.org/fx"
	"go.uber.org/zap"
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
			mongodb.NewMongoStorage,
			localdb.NewLocalStorage,
			repository.NewTemplateRepository,
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
			func() *mongodb.MongoStorageConfig {
				return cfg.MongoConfig
			},
			func() *localdb.LocalStorageConfig {
				return cfg.LocalConfig
			},
		),
		fx.Invoke(func(*server.Server) {}),
		fx.Invoke(func(l *zap.SugaredLogger) {
			l.Infow("Setup complete", "config_path", configPath)
		}),
		fx.Invoke(func(*tracer.Tracer) {}),
		fx.Invoke(func(metrics.MetricsRegistry) {}),
		fx.Invoke(func(*mongodb.MongoStorage) {}),
	), nil
}
