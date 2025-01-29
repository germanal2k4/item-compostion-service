package setup

import (
	"go.uber.org/fx"
	"item_compositiom_service/internal/config"
	"item_compositiom_service/internal/server"
	"item_compositiom_service/internal/services"
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
			func() string {
				return configPath
			},
			func() *server.Config {
				return cfg.GrpcConfig
			},
		),
		fx.Invoke(func(*server.Server) {}),
	), nil
}
