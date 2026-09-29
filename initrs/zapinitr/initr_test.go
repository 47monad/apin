package zapinitr_test

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestShellExposesNativeLoggerIdentity(t *testing.T) {
	shell := newFileShell(t)

	require.NotNil(t, shell.Zap)
	require.NotNil(t, shell.Logger.GetSink())

	underlier, ok := shell.Logger.GetSink().(zapr.Underlier)
	require.True(t, ok, "logr sink should expose the native zap logger")
	require.Same(t, shell.Zap.Core(), underlier.GetUnderlying().Core())
}

func TestLoggerHandlesShareCoreAndFlushOnClose(t *testing.T) {
	shell, output := newFileShellWithOutput(t)

	shell.Logger.Info("through-logr")
	shell.Zap.Info("through-native")

	require.NoError(t, shell.Close(context.Background()))

	contents, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(contents), "through-logr")
	require.Contains(t, string(contents), "through-native")
}

func TestCloseIsIdempotentAndNilSafe(t *testing.T) {
	shell := newFileShell(t)

	require.NoError(t, shell.Close(context.Background()))
	require.NoError(t, shell.Close(context.Background()))

	var nilShell *zapinitr.Shell
	require.NoError(t, nilShell.Close(context.Background()))
	require.NoError(t, (&zapinitr.Shell{}).Close(context.Background()))
}

// syncErrorCore reports a fixed error from Sync to exercise Close's handling of
// non-syncable sinks.
type syncErrorCore struct {
	zapcore.Core
	err error
}

func (core syncErrorCore) Sync() error { return core.err }

func TestCloseToleratesNonSyncableSinks(t *testing.T) {
	for _, err := range []error{syscall.EINVAL, syscall.ENOTTY, syscall.EBADF} {
		shell := &zapinitr.Shell{Zap: zap.New(syncErrorCore{Core: zapcore.NewNopCore(), err: err})}
		require.NoError(t, shell.Close(context.Background()), "errno %v should be benign", err)
	}

	shell := &zapinitr.Shell{Zap: zap.New(syncErrorCore{Core: zapcore.NewNopCore(), err: syscall.EIO})}
	require.ErrorIs(t, shell.Close(context.Background()), syscall.EIO)
}

// newFileShell builds a shell writing to a temp file so Close's native Sync is
// exercised against a regular file.
func newFileShell(t *testing.T) *zapinitr.Shell {
	t.Helper()
	shell, _ := newFileShellWithOutput(t)
	return shell
}

func newFileShellWithOutput(t *testing.T) (*zapinitr.Shell, string) {
	t.Helper()
	output := filepath.Join(t.TempDir(), "zapinitr.log")
	shell, err := zapinitr.New(context.Background(),
		zapinitr.WithLevel("debug"),
		zapinitr.WithNativeConfig(func(config *zap.Config) error {
			config.OutputPaths = []string{output}
			config.ErrorOutputPaths = []string{output}
			return nil
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, shell.Close(context.Background())) })
	return shell, output
}
