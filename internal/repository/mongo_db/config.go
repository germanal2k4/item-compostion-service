package mongodb

import "time"

type MongoStorageConfig struct {
	Enabled                 bool           `yaml:"enable"`
	DSN                     string         `yaml:"dsn"`
	Database                string         `yaml:"database"`
	ClientConfigsCollection string         `yaml:"client_configs_collection"`
	ClientSpecsCollection   string         `yaml:"client_specs_collection"`
	TemplatesCollection     string         `yaml:"templates_collection"`
	OperationTimeout        time.Duration  `yaml:"operation_timeout"`
	ConnectionTimeout       time.Duration  `yaml:"connection_timeout"`
	MaxPoolSize             uint64         `yaml:"max_pool_size"`
	HeartbeatFrequency      time.Duration  `yaml:"heartbeat_frequency"`
	ReadPreference          string         `yaml:"read_preference"`
	LoggingConfig           *LoggingConfig `yaml:"logging"`
}

type LoggingConfig struct {
	Enabled            bool `yaml:"enable"`
	QueryMaxBytesToLog int  `yaml:"query_max_bytes_to_log"`
}
