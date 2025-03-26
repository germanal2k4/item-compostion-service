package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"item_compositiom_service/internal/server"
	"item_compositiom_service/pkg/logger"
	"item_compositiom_service/pkg/tracer"
	"os"
	"time"
)

func ParseConfig(path string) (*Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func GenerateDefaultConfig(configPath string) error {
	defaultCfg, err := convertConfigToMap(getDefaultConfig())
	if err != nil {
		return fmt.Errorf("get default config: %w", err)
	}

	actual := map[string]any{}

	bytes, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(bytes, &actual); err != nil {
			return fmt.Errorf("unmarshal config file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read config file: %w", err)
	}

	res, err := mergeConfig(actual, defaultCfg)
	if err != nil {
		return fmt.Errorf("merge config file: %w", err)
	}

	resBytes, err := yaml.Marshal(res)
	if err != nil {
		return fmt.Errorf("marshal config file: %w", err)
	}

	if err := os.WriteFile(configPath, resBytes, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func getDefaultConfig() *Config {
	return &Config{
		GrpcConfig: &server.Config{
			ListenAddress:  ":8080",
			UnixSocketUser: "",
			Logging: &server.Logging{
				MaxMessageSize: 1024,
			},
			StartDeadline: 5 * time.Second,
			StopDeadline:  5 * time.Second,
		},
		LogConfig: &logger.Config{
			LogLevel:   "debug",
			Transport:  "stdout+file",
			EncodeTime: "RFC3339TimeEncoder",
			DevMode:    true,
			FilePath:   "log/server.log",
		},
		TraceConfig: &tracer.Config{
			Enabled: true,
			Url:     "localhost:4317",
			BatchSpanProcessor: tracer.BatchSpanProcessor{
				MaxQueueSize:       2048,
				MaxExportBatchSize: 512,
				BatchTimeout:       5 * time.Second,
				ExportTimeout:      30 * time.Second,
			},
		},
	}
}

func convertConfigToMap(config *Config) (map[string]interface{}, error) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("serialize config: %w", err)
	}

	res := make(map[string]interface{})
	if err := yaml.Unmarshal(bytes, &res); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return res, nil
}

func mergeConfig(dst map[string]any, src map[string]any) (map[string]any, error) {
	var err error
	for k, v := range src {
		if dst[k] == nil {
			dst[k] = v
		}

		if vv, ok := dst[k].(map[string]any); ok {
			w, ok := v.(map[string]any)
			if !ok {
				continue
			}

			dst[k], err = mergeConfig(vv, w)
			if err != nil {
				return nil, err
			}
		}
	}
	return dst, nil
}
