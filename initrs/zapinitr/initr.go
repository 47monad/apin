package zapinitr

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	config := zap.NewProductionConfig()
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan _2 15:04:05.000000000")
	encoderConfig.StacktraceKey = "" // to hide stacktrace info
	config.EncoderConfig = encoderConfig

	resolved := &resolvedConfig{zapConfig: &config}
	if err := apply(resolved, opts); err != nil {
		return nil, err
	}
	if resolved.levelErr != nil {
		return nil, resolved.levelErr
	}

	zapLog, err := resolved.zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	return &Shell{Logger: zapr.NewLoggerWithOptions(zapLog), zap: zapLog}, nil
}

// Shell exposes the logr logger and retains zap for lifecycle flushing.
type Shell struct {
	Logger logr.Logger
	zap    *zap.Logger
}

// Close flushes buffered zap output. The context is accepted to satisfy the
// common shell lifecycle contract; zap's Sync operation is synchronous.
func (shell *Shell) Close(_ context.Context) error {
	if shell == nil || shell.zap == nil {
		return nil
	}
	if err := shell.zap.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return err
	}
	return nil
}
