package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/rmqinitr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unsetEnv removes key for the duration of the test. An explicit empty process
// value now means an explicit empty configuration, so isolation requires real
// absence rather than t.Setenv(key, "").
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	previous, present := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(key, previous)
			return
		}
		_ = os.Unsetenv(key)
	})
}

type serviceConfig struct {
	Name     string           `json:"name" yaml:"name"`
	RabbitMQ *rmqinitr.Config `json:"rabbitmq" yaml:"rabbitmq"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file, dotenv, process environment", func(t *testing.T) {
		t.Setenv("RABBITMQ_URI", "amqp://process")
		t.Setenv("RABBITMQ_MIN_RETRY_INTERVAL", "12")
		unsetEnv(t, "RABBITMQ_MAX_RETRY_INTERVAL")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name":"json","rabbitmq":{"uri":"amqp://file","minRetryInterval":5,"maxRetryInterval":40}}`), 0o600))
		require.NoError(t, os.WriteFile(envPath, []byte("RABBITMQ_URI=amqp://dotenv\nRABBITMQ_MIN_RETRY_INTERVAL=10\nRABBITMQ_MAX_RETRY_INTERVAL=20\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, envPath, &cfg))
		require.NotNil(t, cfg.RabbitMQ)
		assert.Equal(t, "json", cfg.Name)
		assert.Equal(t, "amqp://process", cfg.RabbitMQ.URI)
		require.NotNil(t, cfg.RabbitMQ.MinRetryInterval)
		assert.Equal(t, 12, *cfg.RabbitMQ.MinRetryInterval)
		require.NotNil(t, cfg.RabbitMQ.MaxRetryInterval)
		assert.Equal(t, 20, *cfg.RabbitMQ.MaxRetryInterval)
	})

	t.Run("YAML", func(t *testing.T) {
		unsetEnv(t, "RABBITMQ_URI")
		unsetEnv(t, "RABBITMQ_MIN_RETRY_INTERVAL")
		unsetEnv(t, "RABBITMQ_MAX_RETRY_INTERVAL")
		configPath := filepath.Join(t.TempDir(), "service.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("name: yaml\nrabbitmq:\n  uri: amqp://yaml\n  minRetryInterval: 3\n  maxRetryInterval: 24\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, "", &cfg))
		require.NotNil(t, cfg.RabbitMQ)
		assert.Equal(t, "yaml", cfg.Name)
		assert.Equal(t, "amqp://yaml", cfg.RabbitMQ.URI)
		require.NotNil(t, cfg.RabbitMQ.MinRetryInterval)
		assert.Equal(t, 3, *cfg.RabbitMQ.MinRetryInterval)
		require.NotNil(t, cfg.RabbitMQ.MaxRetryInterval)
		assert.Equal(t, 24, *cfg.RabbitMQ.MaxRetryInterval)
	})
}
