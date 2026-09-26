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
	// about, and t.Setenv restores it when the subtest ends. Without this,
	// an ambient POSTGRES_* value is overlaid too and its validation can
	// fail these subtests.
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

	t.Run("load_nested_vars/ok", func(t *testing.T) {
		t.Setenv("LOG_LEVEL", "debug")
		t.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		t.Setenv("MONGODB_USERNAME", "testuser")
		t.Setenv("MONGODB_PASSWORD", "testpass")
		t.Setenv("MONGODB_DBNAME", "testdb")
		t.Setenv("POSTGRES_URI", "postgres://localhost:2134")

		cfg := &manifest.Config{
			Name:     "test-app",
			Title:    "Test App",
			Version:  "1.0.0",
			Logging:  manifest.LoggingConfig{},
			Mongodb:  &manifest.MongodbConfig{},
			Postgres: &manifest.PostgresConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Equal(t, "debug", cfg.Logging.Level)
		assert.Equal(t, "mongodb://localhost:27017", cfg.Mongodb.URI)
		assert.Equal(t, "testuser", cfg.Mongodb.Username)
		assert.Equal(t, "testpass", cfg.Mongodb.Password)
		assert.Equal(t, "testdb", cfg.Mongodb.DBName)
		assert.Equal(t, "postgres://localhost:2134", cfg.Postgres.URI)
	})

	// MONGODB_DB_NAME is the standardized name; the legacy MONGODB_DBNAME
	// is covered by load_nested_vars/ok above.
	t.Run("load_new_db_name_var/ok", func(t *testing.T) {
		t.Setenv("MONGODB_DB_NAME", "testdb")

		cfg := &manifest.Config{
			Mongodb: &manifest.MongodbConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Equal(t, "testdb", cfg.Mongodb.DBName)
	})

	t.Run("load_postgres_vars/ok", func(t *testing.T) {
		t.Setenv("POSTGRES_HOST", "localhost")
		t.Setenv("POSTGRES_PORT", "5432")
		t.Setenv("POSTGRES_USERNAME", "postgres")
		t.Setenv("POSTGRES_PASSWORD", "secret")
		t.Setenv("POSTGRES_DB_NAME", "testdb")
		t.Setenv("POSTGRES_SSL_MODE", "require")
		t.Setenv("POSTGRES_APP_NAME", "test-app")
		t.Setenv("POSTGRES_CONN_TIMEOUT", "5")
		t.Setenv("POSTGRES_MODE", "single")
		t.Setenv("POSTGRES_POOL_MAX_CONNS", "10")
		t.Setenv("POSTGRES_POOL_MIN_CONNS", "2")
		t.Setenv("POSTGRES_POOL_MAX_CONN_LIFETIME", "300")
		t.Setenv("POSTGRES_POOL_MAX_CONN_IDLE_TIME", "60")
		t.Setenv("POSTGRES_POOL_HEALTH_CHECK_INTERVAL", "30")

		cfg := &manifest.Config{
			Postgres: &manifest.PostgresConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		pg := cfg.Postgres
		assert.Equal(t, "localhost", pg.Host)
		assert.Equal(t, 5432, pg.Port)
		assert.Equal(t, "postgres", pg.Username)
		assert.Equal(t, "secret", pg.Password)
		assert.Equal(t, "testdb", pg.DBName)
		assert.Equal(t, "require", pg.SSLMode)
		assert.Equal(t, "test-app", pg.AppName)
		assert.Equal(t, 5, pg.ConnTimeout)
		assert.Equal(t, "single", pg.Mode)
		assert.Equal(t, 10, pg.Pool.MaxConns)
		assert.Equal(t, 2, pg.Pool.MinConns)
		assert.Equal(t, 300, pg.Pool.MaxConnLifetime)
		assert.Equal(t, 60, pg.Pool.MaxConnIdleTime)
		assert.Equal(t, 30, pg.Pool.HealthCheckInterval)
	})

	// An optional section absent from the CUE file is allocated when a
	// matching environment variable is present.
	t.Run("postgres_section_allocated_from_env/ok", func(t *testing.T) {
		t.Setenv("POSTGRES_URI", "postgres://localhost:2134/testdb")

		cfg := &manifest.Config{
			// Postgres is nil
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		require.NotNil(t, cfg.Postgres)
		assert.Equal(t, "postgres://localhost:2134/testdb", cfg.Postgres.URI)
	})

	t.Run("postgres_section_not_allocated_without_env/ok", func(t *testing.T) {
		cfg := &manifest.Config{}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Nil(t, cfg.Postgres)
	})

	t.Run("postgres_env_var_validation/error", func(t *testing.T) {
		t.Setenv("POSTGRES_MODE", "garbage")

		cfg := &manifest.Config{
			Postgres: &manifest.PostgresConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode")
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
		t.Setenv("MONGODB_URI", "mongodb://mongo:27017")
		t.Setenv("MONGODB_USERNAME", "produser")
		t.Setenv("MONGODB_PASSWORD", "prodpass")
		t.Setenv("MONGODB_DBNAME", "proddb")
		t.Setenv("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
		t.Setenv("MAIN_GRPC_PORT", "5000")
		t.Setenv("MAIN_HTTP_PORT", "8000")

		cfg := &manifest.Config{
			Name:     "prod-app",
			Title:    "Production App",
			Version:  "1.0.0",
			Logging:  manifest.LoggingConfig{},
			Mongodb:  &manifest.MongodbConfig{},
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
		assert.Equal(t, "mongodb://mongo:27017", cfg.Mongodb.URI)
		assert.Equal(t, "produser", cfg.Mongodb.Username)
		assert.Equal(t, "prodpass", cfg.Mongodb.Password)
		assert.Equal(t, "proddb", cfg.Mongodb.DBName)
		assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/", cfg.RabbitMQ.URI)
		assert.Equal(t, 5000, cfg.GRPC.Servers["main"].Port)
		assert.Equal(t, 8000, cfg.HTTP.Servers["main"].Port)
	})

	t.Run("nil_pointer_in_config/ok", func(t *testing.T) {
		t.Setenv("MONGODB_URI", "mongodb://localhost:27017")

		cfg := &manifest.Config{
			Name:    "test-app",
			Title:   "Test App",
			Version: "1.0.0",
			// Mongodb is nil
		}

		err := manifest.LoadEnvVars(cfg)
		assert.NoError(t, err)
	})
}
