package config

import (
	localdb "item_compositiom_service/internal/repository/local_db"
	mongodb "item_compositiom_service/internal/repository/mongo_db"
	"item_compositiom_service/internal/server"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/tracer"
)

type Config struct {
	GrpcConfig    *server.Config              `yaml:"grpc_server"`
	LogConfig     *logger.Config              `yaml:"logger"`
	TraceConfig   *tracer.Config              `yaml:"trace"`
	MetricsConfig *metrics.Config             `yaml:"metrics"`
	MongoConfig   *mongodb.MongoStorageConfig `yaml:"mongo_storage"`
	LocalConfig   *localdb.LocalStorageConfig `yaml:"local_storage"`
}
