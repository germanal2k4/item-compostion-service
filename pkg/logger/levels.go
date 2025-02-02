package logger

import (
	"context"
	"go.uber.org/zap/zapcore"
)

func Log(ctx context.Context, level zapcore.Level, args ...interface{}) {
	FromContext(ctx).Log(level, args...)
}

func Debug(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Debug(args...)
}

func Info(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Info(args...)
}

func Warn(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Warn(args...)
}

func Error(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Error(args...)
}

func Fatal(ctx context.Context, args ...interface{}) {
	FromContext(ctx).Fatal(args...)
}

func Logf(ctx context.Context, level zapcore.Level, format string, args ...interface{}) {
	FromContext(ctx).Logf(level, format, args...)
}

func Debugf(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Debugf(format, args...)
}

func Infof(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Infof(format, args...)
}

func Warnf(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Warnf(format, args...)
}

func Errorf(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Errorf(format, args...)
}

func Fatalf(ctx context.Context, format string, args ...interface{}) {
	FromContext(ctx).Fatalf(format, args...)
}

func Logw(ctx context.Context, lvl zapcore.Level, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Logw(lvl, msg, keysAndValues...)

}
func Debugw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Debugw(msg, keysAndValues...)
}

func Infow(ctx context.Context, msg string, keysAndVales ...interface{}) {
	FromContext(ctx).Infow(msg, keysAndVales...)
}

func Warnw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Warnw(msg, keysAndValues...)
}

func Errorw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Errorw(msg, keysAndValues...)
}

func Fatalw(ctx context.Context, msg string, keysAndValues ...interface{}) {
	FromContext(ctx).Fatalw(msg, keysAndValues...)
}
