package zapinitr

import (
	"context"
	"fmt"

	"github.com/47monad/apin"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustNew(ctx context.Context, opts ...Option) *apin.LoggerShell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*apin.LoggerShell, error) {
	store := &Store{}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	config := zap.NewProductionConfig()
	if store.Level != "" {
		level, err := zapcore.ParseLevel(store.Level)
		if err != nil {
			return nil, fmt.Errorf("invalid zapinitr log level %q: %w", store.Level, err)
		}
		config.Level = zap.NewAtomicLevelAt(level)
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan _2 15:04:05.000000000")
	encoderConfig.StacktraceKey = "" // to hide stacktrace info
	config.EncoderConfig = encoderConfig

	zapLog, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	return &apin.LoggerShell{
		Logger: zapr.NewLoggerWithOptions(zapLog),
	}, nil
}
