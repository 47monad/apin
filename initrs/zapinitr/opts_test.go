package zapinitr_test

import (
	"context"
	"testing"

	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/stretchr/testify/require"
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

func TestInvalidLevel(t *testing.T) {
	_, err := zapinitr.New(context.Background(), zapinitr.WithLevel("verbose"))
	require.ErrorContains(t, err, `invalid zapinitr log level "verbose"`)
}
