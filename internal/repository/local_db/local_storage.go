package localdb

import (
	"context"
	"item_compositiom_service/pkg/cache"

	"go.uber.org/zap"
)

type LocalStorage struct {
	clientConfigDirPath string
	clientSpecDirPath   string
	templateDirPath     string

	lgr *zap.Logger
	// TODO metrics
}

func NewLocalStorage(config *LocalStorageConfig, lgr *zap.SugaredLogger) *LocalStorage {
	return &LocalStorage{
		clientConfigDirPath: config.ClientConfigDirPath,
		clientSpecDirPath:   config.ClientSpecDirPath,
		templateDirPath:     config.TemplateDirPath,
		lgr:                 lgr.Desugar().With(zap.String("component", "local_storage")),
	}
}

func (s *LocalStorage) UpdateClientConfig(ctx context.Context, setGetter cache.SetGetter[string, any]) error {
	// TODO implement
	return nil
}

func (s *LocalStorage) UpdateClientSpec(ctx context.Context, setGetter cache.SetGetter[string, any]) error {
	// TODO implement
	return nil
}

func (s *LocalStorage) UpdateTemplate(ctx context.Context, setGetter cache.SetGetter[string, any]) error {
	// TODO implement
	return nil
}
