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

	return &Shell{Logger: zapr.NewLoggerWithOptions(zapLog), Zap: zapLog}, nil
}

// Shell exposes the zapinitr logger handles and owns the native logger's
// lifecycle.
type Shell struct {
	// Logger is the logr view of the native zap logger, suitable for
	// components that accept a logr.Logger (for example apin.WithLogger).
	Logger logr.Logger
	// Zap is the native zap logger backing Logger, for zap-specific features
	// such as structured fields or third-party integrations. Close flushes it.
	Zap *zap.Logger
}

// Close flushes buffered zap output from the native logger. The context is
// accepted to satisfy the common shell lifecycle contract; zap's Sync
// operation is synchronous. Sync errors from sinks that cannot be synced
// (stdout/stderr, pipes, terminals) are treated as benign.
func (shell *Shell) Close(_ context.Context) error {
	if shell == nil || shell.Zap == nil {
		return nil
	}
	if err := shell.Zap.Sync(); err != nil && !isSyncBenign(err) {
		return err
	}
	return nil
}

// isSyncBenign reports whether a Sync error comes from a sink that does not
// support syncing. Such errors must not fail shutdown.
func isSyncBenign(err error) bool {
	return errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.ENOTTY) ||
		errors.Is(err, syscall.EBADF)
}
