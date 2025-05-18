package initr

import (
	"context"

	"github.com/47monad/apin/initropts"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Zap = AgentFunc[*initropts.ZapLoggerStore, *LoggerShell](initZapLogger)

func initZapLogger(ctx context.Context, b initropts.Builder[*initropts.ZapLoggerStore]) (*LoggerShell, error) {
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

	zp := zapr.NewLoggerWithOptions(zapLog)

	shell := &LoggerShell{
		Logger: zp,
	}
	return shell, nil
}
