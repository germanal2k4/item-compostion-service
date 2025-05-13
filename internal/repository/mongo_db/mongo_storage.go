package mongodb

import (
	"context"
	"fmt"
	"item_compositiom_service/pkg/cache"
	"item_compositiom_service/pkg/metrics"
	"item_compositiom_service/pkg/tracer"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MongoStorage struct {
	db            *mongo.Database
	ClientConfigs *mongo.Collection
	ClientSpecs   *mongo.Collection
	Templates     *mongo.Collection
	config        *MongoStorageConfig
	cm            *commandMonitor

	lgr *zap.Logger
}

func NewMongoStorage(
	lf fx.Lifecycle,
	config *MongoStorageConfig,
	metrics metrics.MetricsRegistry,
	trace *tracer.Tracer,
	lgr *zap.SugaredLogger,
) (*MongoStorage, error) {
	cm, err := newCommandMonitor(config, metrics, trace)
	if err != nil {
		return nil, fmt.Errorf("create command monitor: %w", err)
	}

	r := &MongoStorage{
		config: config,
		cm:     cm,
		lgr:    lgr.Desugar().With(zap.String("component", component)),
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

func (r *MongoStorage) Start(ctx context.Context) error {
	if !r.config.Enabled {
		r.lgr.Info("Component disabled", zap.String("component", component))
		return nil
	}

	clientOpts := options.Client().ApplyURI(r.config.DSN).
		SetMonitor(provideCommandMonitor(r.cm)).
		SetTimeout(r.config.OperationTimeout).
		SetConnectTimeout(r.config.ConnectionTimeout).
		SetMaxPoolSize(r.config.MaxPoolSize).
		SetHeartbeatInterval(r.config.HeartbeatFrequency)

	switch r.config.ReadPreference {
	case "secondary_preferred":
		clientOpts.SetReadPreference(readpref.SecondaryPreferred())
	case "secondary":
		clientOpts.SetReadPreference(readpref.Secondary())
	case "primary_preferred":
		clientOpts.SetReadPreference(readpref.PrimaryPreferred())
	case "primary":
		clientOpts.SetReadPreference(readpref.Primary())
	default:
		return fmt.Errorf("unexpected read_preference value given: %s", r.config.ReadPreference)
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("connect to mongo: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("%s connection establishment: %w", component, err)
	}

	r.db = client.Database(r.config.Database)
	r.ClientConfigs = r.db.Collection(r.config.ClientConfigsCollection)
	r.ClientSpecs = r.db.Collection(r.config.ClientSpecsCollection)
	r.Templates = r.db.Collection(r.config.TemplatesCollection)

	return nil
}

func (r *MongoStorage) Disconnect(ctx context.Context) error {
	return r.db.Client().Disconnect(ctx)
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

func (s *MongoStorage) UpdateTemplate(ctx context.Context, setGetter cache.SetGetter[string, any]) error {
	// TODO implement
	return nil
}
