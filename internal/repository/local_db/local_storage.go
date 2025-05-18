package localdb

import (
	"context"
	"errors"
	"fmt"
	"item_compositiom_service/internal/entity"
	"item_compositiom_service/pkg/cache"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/parser"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

type LocalStorage struct {
	config *LocalStorageConfig

	templateLib *parser.TemplateLib
	lgr         *zap.Logger
	collector   metricsCollector
}

func NewLocalStorage(config *LocalStorageConfig, lgr *zap.SugaredLogger, m metrics.MetricsRegistry, templateLib *parser.TemplateLib) *LocalStorage {
	collector, err := newMetricsCollector(m.GetRegistry())
	if err != nil {
		lgr.Error("Failed to register metrics for localdb", zap.Error(err))
	}

	return &LocalStorage{
		config:      config,
		templateLib: templateLib,
		lgr:         lgr.Desugar().With(zap.String("component", "local_storage")),
		collector:   *collector,
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

func (s *LocalStorage) UpdateTemplate(ctx context.Context, setGetter cache.SetGetter[entity.TemplateIdName, []parser.Instruction]) error {
	s.collector.updateCount.WithLabelValues("template", "full").Inc()
	dir, err := os.ReadDir(s.config.TemplateDirPath)
	if err != nil {
		s.collector.errorsCount.WithLabelValues("template", "read_dir").Inc()
		return fmt.Errorf("read template dir %s: %w", s.config.TemplateDirPath, err)
	}

	var errs []error

	s.lgr.Debug("LocalStorage find for all templates")

	for _, file := range dir {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if file.IsDir() {
			continue
		}

		name := filepath.Base(file.Name())
		if filepath.Ext(name) != ".yaml" {
			s.lgr.Debug(
				fmt.Sprintf("LocalStorage skip template due unsupported ext: %s", filepath.Ext(name)),
				zap.String("template_id", name),
			)
			continue
		}

		idName := entity.TemplateIdName(strings.TrimSuffix(name, filepath.Ext(name)))

		s.lgr.Debug("LocalStorage template found", zap.String("template_id", string(idName)))

		if lastUpdateTime, ok := setGetter.LastUpdated(idName); ok {
			stat, err := file.Info()
			if err != nil {
				s.collector.errorsCount.WithLabelValues("template", "read_file_stat").Inc()
				errs = append(errs, fmt.Errorf("get template file %s stat: %w", name, err))
				continue
			}

			if stat.ModTime().Before(lastUpdateTime) {
				s.lgr.Debug("LocalStorage template skipped due update time", zap.String("template_id", string(idName)))
				continue
			}
		}

		readTime := time.Now()
		bytes, err := os.ReadFile(file.Name())
		if err != nil {
			s.collector.errorsCount.WithLabelValues("template", "read_file").Inc()
			errs = append(errs, fmt.Errorf("read template file %s: %w", name, err))
			continue
		}
		s.collector.readCount.WithLabelValues("template").Inc()
		s.collector.readDuration.WithLabelValues("template").Observe(time.Since(readTime).Seconds())

		instructions, err := s.templateLib.ParseTemplate(bytes)
		if err != nil {
			s.collector.errorsCount.WithLabelValues("template", "parse_template").Inc()
			errs = append(errs, fmt.Errorf("parse template %s: %w", name, err))
			continue
		}

		setGetter.Set(idName, instructions, readTime)
	}

	return errors.Join(errs...)
}

func (s *LocalStorage) IncrementalUpdateTemplate(ctx context.Context, setGetter cache.SetGetter[entity.TemplateIdName, []parser.Instruction], id entity.TemplateIdName) error {
	s.collector.updateCount.WithLabelValues("template", "incremental").Inc()
	dir, err := os.ReadDir(s.config.TemplateDirPath)
	if err != nil {
		s.collector.errorsCount.WithLabelValues("template", "read_dir").Inc()
		return fmt.Errorf("read template dir %s: %w", s.config.TemplateDirPath, err)
	}

	s.lgr.Debug("LocalStorage find for a template", zap.String("template_id", string(id)))

	var errs []error

	for _, file := range dir {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if file.IsDir() {
			continue
		}

		name := file.Name()
		idName := entity.TemplateIdName(strings.TrimSuffix(name, filepath.Ext(name)))
		if idName != id {
			continue
		}

		s.lgr.Debug("LocalStorage template found", zap.String("template_id", string(idName)))

		readTime := time.Now()
		bytes, err := os.ReadFile(file.Name())
		if err != nil {
			s.collector.errorsCount.WithLabelValues("template", "read_file").Inc()
			errs = append(errs, fmt.Errorf("read template file %s: %w", name, err))
			continue
		}
		s.collector.readCount.WithLabelValues("template").Inc()
		s.collector.readDuration.WithLabelValues("template").Observe(time.Since(readTime).Seconds())

		instructions, err := s.templateLib.ParseTemplate(bytes)
		if err != nil {
			s.collector.errorsCount.WithLabelValues("template", "parse_template").Inc()
			errs = append(errs, fmt.Errorf("parse template %s: %w", name, err))
			continue
		}

		setGetter.Set(idName, instructions, readTime)
	}

	return errors.Join(errs...)
}

func (s *LocalStorage) LogDebug(msg string, fields ...zap.Field) {
	if s.config.LoggingConfig.Enabled {
		s.lgr.Debug(msg, fields...)
	}
}
