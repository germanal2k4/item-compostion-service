package logger

import (
	"context"
	"go.uber.org/zap"
)

type contextKey struct{}

var (
	loggerContextKey = contextKey{}
)

func ToContext(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

func ContextWithKV(ctx context.Context, kvs ...interface{}) context.Context {
	l := FromContext(ctx).Desugar()
	result := make([]zap.Field, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		if i == len(kvs)-1 {
			break
		}

		key, ok := kvs[i].(string)
		if !ok {
			l.Warn("key is not a string", zap.Any("key", kvs[i+1]))
			continue
		}

		result = append(result, zap.Any(key, kvs[i+1]))
	}

	return ToContext(ctx, l.With(result...).Sugar())
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	l, ok := ctx.Value(loggerContextKey).(*zap.SugaredLogger)
	if !ok {
		return fallbackLogger
	}
	return l
}
