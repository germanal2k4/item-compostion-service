package metrics

type Config struct {
	Enable bool `yaml:"enable"`
	Port   int  `yaml:"port"`
}
