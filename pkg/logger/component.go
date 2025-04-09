package logger

import (
	"context"
	"fmt"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	fallbackLogger *zap.SugaredLogger
)

func init() {
	fallbackLogger = newFallBackLogger()
}

func FallbackLogger() *zap.SugaredLogger {
	return fallbackLogger
}

func NewLogger(lc fx.Lifecycle, cfg *Config) (*zap.SugaredLogger, error) {
	if cfg == nil || cfg.LogLevel == "" || cfg.Transport == "" {
		return nil, fmt.Errorf("invalid logger configuration, required: log_level, transport")
	}

	var (
		err   error
		cores []zapcore.Core
		stops []func()
		info  *loggerInfo
	)

	if info, err = enrichLoggerInfo(cfg); err != nil {
		return nil, err
	}

	if cfg.Transport == stdoutTransport {
		stdoutTransport := getStdoutTransport(info)
		cores = append(cores, stdoutTransport.core)
		stops = append(stops, stdoutTransport.stop)
	}

	if cfg.Transport == fileTransport || cfg.Transport == fileAndElasticTransport {
		fileTransport, err := getFileTransport(info)
		if err != nil {
			return nil, fmt.Errorf("failed to get file transport for logger: %w", err)
		}

		cores = append(cores, fileTransport.core)
		stops = append(stops, fileTransport.stop)
	}

	if cfg.Transport == fileAndElasticTransport {
		elasticTransport, err := getElasticTransport(info)
		if err != nil {
			return nil, fmt.Errorf("failed to get elastic transport for logger: %w", err)
		}

		cores = append(cores, elasticTransport.core)
		stops = append(stops, elasticTransport.stop)
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no logger could be created for %s", cfg.Transport)
	}

	lgr := zap.New(zapcore.NewTee(cores...), info.opts...).Sugar()

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			for _, stop := range stops {
				stop()
			}
			return nil
		},
	})

	return lgr, nil
}

func newFallBackLogger() *zap.SugaredLogger {
	fallbackCfg := &Config{
		LogLevel:   "info",
		Transport:  stdoutTransport,
		EncodeTime: "ISO8601TimeEncoder",
		DevMode:    true,
	}

	info, err := enrichLoggerInfo(fallbackCfg)

	if err != nil {
		panic(err)
	}

	stdoutTransport := getStdoutTransport(info)
	lgr := zap.New(stdoutTransport.core).Sugar()
	return lgr
}

type loggerInfo struct {
	cfg    *Config
	encCfg zapcore.EncoderConfig
	lvl    zap.AtomicLevel
	opts   []zap.Option
}

func enrichLoggerInfo(cfg *Config) (*loggerInfo, error) {
	info := &loggerInfo{
		cfg:  cfg,
		opts: []zap.Option{},
	}

	info.encCfg = zap.NewProductionEncoderConfig()
	switch cfg.EncodeTime {
	case "RFC3339TimeEncoder":
		info.encCfg.EncodeTime = zapcore.RFC3339TimeEncoder
	default:
		info.encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	info.lvl = zap.NewAtomicLevel()
	if err := info.lvl.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, err
	}

	return info, nil
}
