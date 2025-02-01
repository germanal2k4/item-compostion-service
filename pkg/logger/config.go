package logger

type Config struct {
	LogLevel   string `yaml:"log_level"`
	FilePath   string `yaml:"file_path"`
	Transport  string `yaml:"transport"`
	EncodeTime string `yaml:"encode_time"`
	// EnableAsync bool   `yaml:"enable_async"`
	DevMode bool `yaml:"dev_mode"`
}
