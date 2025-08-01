package zapinitr

import (
	"context"

	"github.com/47monad/apin"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustNew(ctx context.Context, b apin.Builder[*Store]) *apin.LoggerShell {
	shell, err := _init(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, b apin.Builder[*Store]) (*apin.LoggerShell, error) {
	return _init(ctx, b)
}

func _init(ctx context.Context, b apin.Builder[*Store]) (*apin.LoggerShell, error) {
	_, err := b.Build()
	if err != nil {
		return nil, err
	}
	config := zap.NewProductionConfig()
	encoderConfig := zap.NewProductionEncoderConfig()
	zapcore.TimeEncoderOfLayout("Jan _2 15:04:05.000000000")
	encoderConfig.StacktraceKey = "" // to hide stacktrace info
	config.EncoderConfig = encoderConfig

	zapLog, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return &apin.LoggerShell{
		Logger: zapr.NewLoggerWithOptions(zapLog),
	}, nil
}
