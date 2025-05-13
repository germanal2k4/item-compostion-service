package localdb

type LocalStorageConfig struct {
	ClientConfigDirPath string `yaml:"client_config_dir_path"`
	ClientSpecDirPath   string `yaml:"client_spec_dir_path"`
	TemplateDirPath     string `yaml:"template_dir_path"`
}
