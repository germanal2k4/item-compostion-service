package config

import (
	"item_compositiom_service/internal/server"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/tracer"
)

type Config struct {
	GrpcConfig  *server.Config `yaml:"grpc_server"`
	LogConfig   *logger.Config `yaml:"logger"`
	TraceConfig *tracer.Config `yaml:"trace"`
}
