package config

import (
	"item_compositiom_service/internal/server"
)

type Config struct {
	GrpcConfig *server.Config `yaml:"grpc_server"`
}
