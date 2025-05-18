package mongodb

import (
	"context"
	"errors"
	"fmt"
	"item_compositiom_service/internal/entity"
	"item_compositiom_service/pkg/cache"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/parser"
	"item_compositiom_service/pkg/tracer"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gopkg.in/mgo.v2/bson"
)

type MongoStorage struct {
	db            *mongo.Database
	ClientConfigs *mongo.Collection
	ClientSpecs   *mongo.Collection
	Templates     *mongo.Collection
	config        *MongoStorageConfig
	cm            *commandMonitor

	templateLib *parser.TemplateLib
	lgr         *zap.Logger
}

func NewMongoStorage(
	lf fx.Lifecycle,
	config *MongoStorageConfig,
	metrics metrics.MetricsRegistry,
	trace *tracer.Tracer,
	lgr *zap.SugaredLogger,
	templateLib *parser.TemplateLib,
) (*MongoStorage, error) {
	cm, err := newCommandMonitor(config, metrics, trace)
	if err != nil {
		return nil, fmt.Errorf("create command monitor: %w", err)
	}

	r := &MongoStorage{
		config:      config,
		cm:          cm,
		templateLib: templateLib,
		lgr:         lgr.Desugar().With(zap.String("component", component)),
	}

	lf.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return r.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return r.Disconnect(ctx)
		},
	})

	return r, nil
}

func (s *MongoStorage) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.lgr.Info("Component disabled")
		return nil
	}

	clientOpts := options.Client().ApplyURI(s.config.DSN).
		SetMonitor(provideCommandMonitor(s.cm)).
		SetTimeout(s.config.OperationTimeout).
		SetConnectTimeout(s.config.ConnectionTimeout).
		SetMaxPoolSize(s.config.MaxPoolSize).
		SetHeartbeatInterval(s.config.HeartbeatFrequency)

	switch s.config.ReadPreference {
	case "secondary_preferred":
		clientOpts.SetReadPreference(readpref.SecondaryPreferred())
	case "secondary":
		clientOpts.SetReadPreference(readpref.Secondary())
	case "primary_preferred":
		clientOpts.SetReadPreference(readpref.PrimaryPreferred())
	case "primary":
		clientOpts.SetReadPreference(readpref.Primary())
	default:
		return fmt.Errorf("unexpected read_preference value given: %s", s.config.ReadPreference)
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("connect to mongo: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("%s connection establishment: %w", component, err)
	}

	s.db = client.Database(s.config.Database)
	s.ClientConfigs = s.db.Collection(s.config.ClientConfigsCollection)
	s.ClientSpecs = s.db.Collection(s.config.ClientSpecsCollection)
	s.Templates = s.db.Collection(s.config.TemplatesCollection)

	if err := s.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}

	s.lgr.Info("Component started")

	return nil
}

func (s *MongoStorage) Disconnect(ctx context.Context) error {
	return s.db.Client().Disconnect(ctx)
}

func (s *MongoStorage) CreateIndexes(ctx context.Context) error {
	_, err := s.Templates.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"template_view_id": 1},
	})
	return err
}

// do not forbid to propagate logger to context for command monitor
func (s *MongoStorage) UpdateClientConfig(ctx context.Context, setGetter cache.SetGetter[string, any]) error {
	// TODO implement
	return nil
}

func (s *MongoStorage) UpdateClientSpec(ctx context.Context, setGetter cache.SetGetter[string, any]) error {
	// TODO implement
	return nil
}

type mongoTemplate struct {
	ID        string    `bson:"template_view_id"`
	Content   []byte    `bson:"content"`
	UpdatedAt time.Time `bson:"updated_at"`
}

func (s *MongoStorage) UpdateTemplate(ctx context.Context, setGetter cache.SetGetter[entity.TemplateIdName, []parser.Instruction]) error {
	readTime := time.Now()
	ctx = WithCommandName(ctx, "FindTemplateList")
	cursor, err := s.Templates.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to find configs: %w", err)
	}
	defer cursor.Close(ctx)

	var errs []error
	var nullTime time.Time

	for cursor.Next(ctx) {
		var template mongoTemplate
		if err := cursor.Decode(&template); err != nil {
			errs = append(errs, fmt.Errorf("failed to decode config: %w", err))
			continue
		}

		if template.UpdatedAt == nullTime {
			template.UpdatedAt = time.Now()
		}

		if lastUpdatedTime, ok := setGetter.LastUpdated(entity.TemplateIdName(template.ID)); ok && template.UpdatedAt.Before(lastUpdatedTime) {
			if s.config.LoggingConfig.Enabled {
				s.lgr.Debug("MongoStorage template skipped due update time", zap.String("template_view_id", template.ID))
			}
			continue
		}

		instructions, err := s.templateLib.ParseTemplate(template.Content)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse template: %w", err))
			continue
		}
		setGetter.Set(entity.TemplateIdName(template.ID), instructions, readTime)
	}

	if err := cursor.Err(); err != nil {
		errs = append(errs, fmt.Errorf("cursor error: %w", err))
	}

	return errors.Join(errs...)
}

func (s *MongoStorage) IncrementalUpdateTemplate(ctx context.Context, setGetter cache.SetGetter[entity.TemplateIdName, []parser.Instruction], id entity.TemplateIdName) error {
	var result mongoTemplate
	readTime := time.Now()

	ctx = WithCommandName(ctx, "FindTemplate")
	err := s.Templates.FindOne(ctx, bson.M{"template_view_id": id}).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("find template in mongo: template %s not found: %w", id, err)
		}
		return fmt.Errorf("find template in mongo: %w", err)
	}

	instructions, err := s.templateLib.ParseTemplate(result.Content)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	setGetter.Set(id, instructions, readTime)

	return nil
}
