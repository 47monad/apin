package zapinitr_test

import (
	"context"
	"errors"
	"testing"

	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestConfigAndOptionPrecedence(t *testing.T) {
	config := &zapinitr.Config{Level: "debug"}
	shell, err := zapinitr.New(context.Background(),
		zapinitr.WithConfig(config),
		zapinitr.WithLevel("error"),
	)
	require.NoError(t, err)
	require.NotNil(t, shell)
	require.NotNil(t, shell.Logger.GetSink())
	require.NoError(t, shell.Close(context.Background()))
}

func TestOptionsOnlyConstruction(t *testing.T) {
	shell, err := zapinitr.New(context.Background(), zapinitr.WithLevel("debug"))
	require.NoError(t, err)
	require.NotNil(t, shell)
	require.NoError(t, shell.Close(context.Background()))
}

func TestValidationOccursAfterAllOptions(t *testing.T) {
	shell, err := zapinitr.New(context.Background(),
		zapinitr.WithLevel("invalid"),
		zapinitr.WithLevel("debug"),
	)
	require.NoError(t, err)
	require.NoError(t, shell.Close(context.Background()))

	shell, err = zapinitr.New(context.Background(),
		zapinitr.WithLevel("invalid"),
		zapinitr.WithNativeConfig(func(config *zap.Config) error {
			config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
			return nil
		}),
	)
	require.NoError(t, err)
	require.NoError(t, shell.Close(context.Background()))
}

func TestNativeConfigEscapeHatch(t *testing.T) {
	stop := errors.New("inspect native zap config")
	called := false
	_, err := zapinitr.New(context.Background(),
		zapinitr.WithLevel("debug"),
		zapinitr.WithNativeConfig(func(config *zap.Config) error {
			called = true
			require.Equal(t, zapcore.DebugLevel, config.Level.Level())
			require.Empty(t, config.EncoderConfig.StacktraceKey)
			config.Development = true
			return stop
		}),
	)
	require.ErrorIs(t, err, stop)
	require.True(t, called)
}

func TestInvalidLevel(t *testing.T) {
	_, err := zapinitr.New(context.Background(), zapinitr.WithLevel("verbose"))
	require.ErrorContains(t, err, `invalid zapinitr log level "verbose"`)
}
