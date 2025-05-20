package repository

import (
	"context"
	"errors"
	"item_compositiom_service/internal/entity"
	localdb "item_compositiom_service/internal/repository/local_db"
	mongodb "item_compositiom_service/internal/repository/mongo_db"
	"item_compositiom_service/pkg/cache"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/parser"

	"go.uber.org/zap"
)

type TemplateRepository struct {
	ms *mongodb.MongoStorage
	ls *localdb.LocalStorage

	cache *cache.Cache[entity.TemplateIdName, []parser.Instruction]
}

func NewTemplateRepository(
	logger *zap.SugaredLogger,
	metrics metrics.MetricsRegistry,
	ms *mongodb.MongoStorage,
	ls *localdb.LocalStorage,
) *TemplateRepository {
	cache := cache.New(
		logger,
		metrics,
		func(ctx context.Context, sg cache.SetGetter[entity.TemplateIdName, []parser.Instruction]) error {
			if err := ms.UpdateTemplate(ctx, sg); err != nil {
				if err2 := ls.UpdateTemplate(ctx, sg); err2 != nil {
					return errors.Join(err, err2)
				}

				return err
			}

			return nil
		},
		func(ctx context.Context, sg cache.SetGetter[entity.TemplateIdName, []parser.Instruction], key entity.TemplateIdName) error {
			if err := ms.IncrementalUpdateTemplate(ctx, sg, key); err != nil {
				if err2 := ls.IncrementalUpdateTemplate(ctx, sg, key); err2 != nil {
					return errors.Join(err, err2)
				}

				return err
			}

			return nil
		},
	)

	return &TemplateRepository{
		ms:    ms,
		ls:    ls,
		cache: cache,
	}
}

func (r *TemplateRepository) GetTemplate(key entity.TemplateIdName) ([]parser.Instruction, bool) {
	return r.cache.Get(key)
}

func (r *TemplateRepository) UpdateTemplate(ctx context.Context, key entity.TemplateIdName) {
	r.cache.IncrementalUpdate(ctx, key)
}
