package server

import (
	"fmt"
	"time"
)

type Config struct {
	ListenAddress  string        `yaml:"listen_address"`
	Logging        *Logging      `yaml:"logging"`
	UnixSocketUser string        `yaml:"unix_socket_user"`
	StartDeadline  time.Duration `yaml:"start_deadline"`
	StopDeadline   time.Duration `yaml:"stop_deadline"`
}

type Logging struct {
	Disable                   bool   `yaml:"disable"`
	DisableEnrichTraces       bool   `yaml:"disable_enrich_traces"`
	DisableLogRequestMessage  bool   `yaml:"disable_log_request_message"`
	DisableLogResponseMessage bool   `yaml:"disable_log_response_message"`
	LogLevel                  string `yaml:"log_level"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	tmp := struct {
		ListenAddress  *string        `yaml:"listen_address"`
		Logging        *Logging       `yaml:"logging"`
		UnixSocketUser string         `yaml:"unix_socket_user"`
		StartDeadline  *time.Duration `yaml:"start_deadline"`
		StopDeadline   *time.Duration `yaml:"stop_deadline"`
	}{}

	if err := unmarshal(&tmp); err != nil {
		return err
	}

	if tmp.ListenAddress == nil {
		return fmt.Errorf("missing requred `grpc_server.listen_address`")
	}

	if tmp.StartDeadline == nil {
		return fmt.Errorf("missing requred `grpc_server.start_deadline`")
	}

	if tmp.StopDeadline == nil {
		return fmt.Errorf("missing requred `grpc_server.stop_deadline`")
	}

	if tmp.Logging != nil || tmp.Logging.LogLevel == "" {
		return fmt.Errorf("missing required `grpc_server.log_level`")
	}

	c.ListenAddress = *tmp.ListenAddress
	c.UnixSocketUser = tmp.UnixSocketUser
	c.Logging = tmp.Logging
	c.StartDeadline = *tmp.StartDeadline
	c.StopDeadline = *tmp.StopDeadline

	return nil
}
