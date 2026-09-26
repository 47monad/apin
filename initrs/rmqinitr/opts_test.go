package rmqinitr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultsAndOptionPrecedence(t *testing.T) {
	shell, err := New(t.Context(), WithURI("amqp://default"), WithLazyConnect())
	require.NoError(t, err)
	defer shell.Close(t.Context())
	assert.Equal(t, defaultMinRetryInterval, shell.store.MinRetryInterval)
	assert.Equal(t, defaultMaxRetryInterval, shell.store.MaxRetryInterval)

	minimum, maximum := 4, 12
	configured, err := New(t.Context(),
		WithConfig(&Config{URI: "amqp://config", MinRetryInterval: &minimum, MaxRetryInterval: &maximum}),
		WithURI("amqp://first"),
		WithURI("amqp://second"),
		WithMinRetryInterval(2*time.Second),
		WithLazyConnect(),
	)
	require.NoError(t, err)
	defer configured.Close(t.Context())
	assert.Equal(t, "amqp://second", configured.store.URI)
	assert.Equal(t, 2*time.Second, configured.store.MinRetryInterval)
	assert.Equal(t, 12*time.Second, configured.store.MaxRetryInterval)
}

func TestWithConfigValidationAndOverrides(t *testing.T) {
	zero, one, four, five := 0, 1, 4, 5
	for _, tc := range []struct {
		name   string
		config *Config
		want   string
	}{
		{name: "minimum must be positive", config: &Config{URI: "amqp://localhost", MinRetryInterval: &zero}, want: "min retry interval"},
		{name: "maximum must exceed one second", config: &Config{URI: "amqp://localhost", MaxRetryInterval: &one}, want: "max retry interval"},
		{name: "maximum must not be lower than minimum", config: &Config{URI: "amqp://localhost", MinRetryInterval: &five, MaxRetryInterval: &four}, want: "lower than min"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(t.Context(), WithConfig(tc.config), WithLazyConnect())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestNewRequiresURI(t *testing.T) {
	_, err := New(t.Context(), WithLazyConnect())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no rabbitmq configuration")
}
