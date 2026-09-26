package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreEnv puts name back to its pre-test state when t finishes. It exists
// only for the deprecated LoadEnvFile, which writes to the process
// environment behind t's back; every other variable in this package is set
// with t.Setenv, which restores automatically and needs no help.
func restoreEnv(t *testing.T, name string) {
	t.Helper()
	original, existed := os.LookupEnv(name)
	t.Cleanup(func() {
		if existed {
			os.Setenv(name, original)
			return
		}
		os.Unsetenv(name)
	})
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.env")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadEnvFile(t *testing.T) {
	// LoadEnvFile is deprecated: it is the only remaining entry point that
	// mutates the process environment. This test pins that behavior so the
	// deprecation stays honest — see issue #41 for the reasoning.
	t.Run("nonexistent_env/error", func(t *testing.T) {
		err := manifest.LoadEnvFile("nonexistent.env")
		assert.Error(t, err)
	})

	t.Run("load_env/ok", func(t *testing.T) {
		restoreEnv(t, "TEST_ENV")

		path := writeEnvFile(t, "TEST_ENV=test\n")

		err := manifest.LoadEnvFile(path)
		assert.NoError(t, err)

		assert.Equal(t, "test", os.Getenv("TEST_ENV"))
	})

	// godotenv never overwrites a variable that is already set, which is why
	// an exported value keeps winning over a .env entry.
	t.Run("existing_value_is_kept/ok", func(t *testing.T) {
		t.Setenv("TEST_ENV", "from-process")

		path := writeEnvFile(t, "TEST_ENV=from-file\n")

		err := manifest.LoadEnvFile(path)
		assert.NoError(t, err)

		assert.Equal(t, "from-process", os.Getenv("TEST_ENV"))
	})
}

func TestLoadEnvVars(t *testing.T) {
	// LoadEnvVars reads the process environment, so shield the whole tree
	// from the developer's own shell. Each subtest then sets only what it is
	// about, and t.Setenv restores it when the subtest ends.
	clearEnv(t)

	t.Run("nil_config/error", func(t *testing.T) {
		err := manifest.LoadEnvVars(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("load_basic_vars/ok", func(t *testing.T) {
		t.Setenv("ENV", "test")
		t.Setenv("MODE", "debug")
		t.Setenv("HOST", "localhost")
		t.Setenv("LOG_LEVEL", "info")

		cfg := &manifest.Config{
			Name:    "test-app",
			Title:   "Test App",
			Version: "1.0.0",
			Logging: manifest.LoggingConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Equal(t, "test", cfg.Env)
		assert.Equal(t, "debug", cfg.Mode)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, "info", cfg.Logging.Level)
	})

	t.Run("load_numeric_vars/ok", func(t *testing.T) {
		t.Setenv("MAIN_GRPC_PORT", "50051")
		t.Setenv("MAIN_HTTP_PORT", "8080")

		cfg := &manifest.Config{
			Name:    "test-app",
			Title:   "Test App",
			Version: "1.0.0",
			GRPC:    &manifest.GRPCConfig{Servers: map[string]manifest.GRPCServerConfig{"main": {}}},
			HTTP:    &manifest.HTTPConfig{Servers: map[string]manifest.HTTPServerConfig{"main": {}}},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Equal(t, 50051, cfg.GRPC.Servers["main"].Port)
		assert.Equal(t, 8080, cfg.HTTP.Servers["main"].Port)
	})

	t.Run("invalid_numeric_var/error", func(t *testing.T) {
		t.Setenv("MAIN_HTTP_PORT", "not-a-number")

		cfg := &manifest.Config{
			HTTP: &manifest.HTTPConfig{Servers: map[string]manifest.HTTPServerConfig{"main": {}}},
		}

		err := manifest.LoadEnvVars(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error converting env var")
	})

	// TODO: Test Boolean vars. currently no boolean env var exist

	t.Run("map_fields/ok", func(t *testing.T) {
		t.Setenv("SERVICE1_GRPC_CLIENT_ADDRESS", "localhost:50051")

		cfg := &manifest.Config{
			GRPC: &manifest.GRPCConfig{
				Clients: map[string]manifest.GRPCClientConfig{
					"service1": {},
				},
			},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)
		assert.Equal(t, "localhost:50051", cfg.GRPC.Clients["service1"].Address)
	})

	t.Run("full/ok", func(t *testing.T) {
		t.Setenv("ENV", "production")
		t.Setenv("MODE", "release")
		t.Setenv("HOST", "0.0.0.0")
		t.Setenv("LOG_LEVEL", "info")
		t.Setenv("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
		t.Setenv("MAIN_GRPC_PORT", "5000")
		t.Setenv("MAIN_HTTP_PORT", "8000")

		cfg := &manifest.Config{
			Name:     "prod-app",
			Title:    "Production App",
			Version:  "1.0.0",
			Logging:  manifest.LoggingConfig{},
			RabbitMQ: &manifest.RabbitMQConfig{},
			GRPC:     &manifest.GRPCConfig{Servers: map[string]manifest.GRPCServerConfig{"main": {}}},
			HTTP:     &manifest.HTTPConfig{Servers: map[string]manifest.HTTPServerConfig{"main": {}}},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Equal(t, "production", cfg.Env)
		assert.Equal(t, "release", cfg.Mode)
		assert.Equal(t, "0.0.0.0", cfg.Host)
		assert.Equal(t, "info", cfg.Logging.Level)
		assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/", cfg.RabbitMQ.URI)
		assert.Equal(t, 5000, cfg.GRPC.Servers["main"].Port)
		assert.Equal(t, 8000, cfg.HTTP.Servers["main"].Port)
	})

	t.Run("nil_pointer_in_config/ok", func(t *testing.T) {
		cfg := &manifest.Config{
			Name:    "test-app",
			Title:   "Test App",
			Version: "1.0.0",
		}

		err := manifest.LoadEnvVars(cfg)
		assert.NoError(t, err)
	})
}
