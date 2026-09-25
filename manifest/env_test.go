package manifest_test

import (
	"os"
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile(t *testing.T) {
	t.Run("nonexistent_env/error", func(t *testing.T) {
		err := manifest.LoadEnvFile("nonexistent.env")
		assert.Error(t, err)
	})

	// Create a temporary .env file for testing
	t.Run("load_env/ok", func(t *testing.T) {
		content := `
		TEST_ENV=test
		`
		tmpFile, err := os.CreateTemp("", "test*.env")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		tmpFile.Close()

		err = manifest.LoadEnvFile(tmpFile.Name())
		assert.NoError(t, err)

		assert.Equal(t, "test", os.Getenv("TEST_ENV"))

		os.Unsetenv("TEST_ENV")
	})
}

func TestLoadEnvVars(t *testing.T) {
	t.Run("nil_config/error", func(t *testing.T) {
		err := manifest.LoadEnvVars(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	// Setup helper function to reset env vars after each test
	resetEnvVars := func() {
		os.Unsetenv("ENV")
		os.Unsetenv("MODE")
		os.Unsetenv("HOST")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("MONGODB_URI")
		os.Unsetenv("MONGODB_USERNAME")
		os.Unsetenv("MONGODB_PASSWORD")
		os.Unsetenv("MONGODB_DBNAME")
		os.Unsetenv("MONGODB_DB_NAME")
		os.Unsetenv("POSTGRES_URI")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USERNAME")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB_NAME")
		os.Unsetenv("POSTGRES_SSL_MODE")
		os.Unsetenv("POSTGRES_APP_NAME")
		os.Unsetenv("POSTGRES_CONN_TIMEOUT")
		os.Unsetenv("POSTGRES_MODE")
		os.Unsetenv("POSTGRES_POOL_MAX_CONNS")
		os.Unsetenv("POSTGRES_POOL_MIN_CONNS")
		os.Unsetenv("POSTGRES_POOL_MAX_CONN_LIFETIME")
		os.Unsetenv("POSTGRES_POOL_MAX_CONN_IDLE_TIME")
		os.Unsetenv("POSTGRES_POOL_HEALTH_CHECK_INTERVAL")
		os.Unsetenv("RABBITMQ_URI")
		os.Unsetenv("MAIN_GRPC_PORT")
		os.Unsetenv("MAIN_GRPC_CLIENT_ADDRESS")
		os.Unsetenv("MAIN_HTTP_PORT")
	}

	t.Run("load_basic_vars/ok", func(t *testing.T) {
		defer resetEnvVars()

		os.Setenv("ENV", "test")
		os.Setenv("MODE", "debug")
		os.Setenv("HOST", "localhost")
		os.Setenv("LOG_LEVEL", "info")

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
		defer resetEnvVars()

		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
		os.Setenv("MONGODB_USERNAME", "testuser")
		os.Setenv("MONGODB_PASSWORD", "testpass")
		os.Setenv("MONGODB_DBNAME", "testdb")
		os.Setenv("POSTGRES_URI", "postgres://localhost:2134")

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
		defer resetEnvVars()

		os.Setenv("MONGODB_DB_NAME", "testdb")

		cfg := &manifest.Config{
			Mongodb: &manifest.MongodbConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Equal(t, "testdb", cfg.Mongodb.DBName)
	})

	t.Run("load_postgres_vars/ok", func(t *testing.T) {
		defer resetEnvVars()

		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USERNAME", "postgres")
		os.Setenv("POSTGRES_PASSWORD", "secret")
		os.Setenv("POSTGRES_DB_NAME", "testdb")
		os.Setenv("POSTGRES_SSL_MODE", "require")
		os.Setenv("POSTGRES_APP_NAME", "test-app")
		os.Setenv("POSTGRES_CONN_TIMEOUT", "5")
		os.Setenv("POSTGRES_MODE", "single")
		os.Setenv("POSTGRES_POOL_MAX_CONNS", "10")
		os.Setenv("POSTGRES_POOL_MIN_CONNS", "2")
		os.Setenv("POSTGRES_POOL_MAX_CONN_LIFETIME", "300")
		os.Setenv("POSTGRES_POOL_MAX_CONN_IDLE_TIME", "60")
		os.Setenv("POSTGRES_POOL_HEALTH_CHECK_INTERVAL", "30")

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
		defer resetEnvVars()

		os.Setenv("POSTGRES_URI", "postgres://localhost:2134/testdb")

		cfg := &manifest.Config{
			// Postgres is nil
		}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		require.NotNil(t, cfg.Postgres)
		assert.Equal(t, "postgres://localhost:2134/testdb", cfg.Postgres.URI)
	})

	t.Run("postgres_section_not_allocated_without_env/ok", func(t *testing.T) {
		defer resetEnvVars()

		cfg := &manifest.Config{}

		err := manifest.LoadEnvVars(cfg)
		require.NoError(t, err)

		assert.Nil(t, cfg.Postgres)
	})

	t.Run("postgres_env_var_validation/error", func(t *testing.T) {
		defer resetEnvVars()

		os.Setenv("POSTGRES_MODE", "garbage")

		cfg := &manifest.Config{
			Postgres: &manifest.PostgresConfig{},
		}

		err := manifest.LoadEnvVars(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode")
	})

	t.Run("load_numeric_vars/ok", func(t *testing.T) {
		defer resetEnvVars()

		os.Setenv("MAIN_GRPC_PORT", "50051")
		os.Setenv("MAIN_HTTP_PORT", "8080")

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
		defer resetEnvVars()

		os.Setenv("MAIN_HTTP_PORT", "not-a-number")

		cfg := &manifest.Config{
			HTTP: &manifest.HTTPConfig{Servers: map[string]manifest.HTTPServerConfig{"main": {}}},
		}

		err := manifest.LoadEnvVars(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error converting env var")
	})

	// TODO: Test Boolean vars. currently no boolean env var exist

	t.Run("map_fields/ok", func(t *testing.T) {
		defer resetEnvVars()

		os.Setenv("SERVICE1_GRPC_CLIENT_ADDRESS", "localhost:50051")

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
		defer resetEnvVars()

		os.Setenv("ENV", "production")
		os.Setenv("MODE", "release")
		os.Setenv("HOST", "0.0.0.0")
		os.Setenv("LOG_LEVEL", "info")
		os.Setenv("MONGODB_URI", "mongodb://mongo:27017")
		os.Setenv("MONGODB_USERNAME", "produser")
		os.Setenv("MONGODB_PASSWORD", "prodpass")
		os.Setenv("MONGODB_DBNAME", "proddb")
		os.Setenv("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
		os.Setenv("MAIN_GRPC_PORT", "5000")
		os.Setenv("MAIN_HTTP_PORT", "8000")

		cfg := &manifest.Config{
			Name:    "prod-app",
			Title:   "Production App",
			Version: "1.0.0",
			Logging: manifest.LoggingConfig{},
			Mongodb: &manifest.MongodbConfig{},
			RabbiMQ: &manifest.RabbitMQConfig{},
			GRPC:    &manifest.GRPCConfig{Servers: map[string]manifest.GRPCServerConfig{"main": {}}},
			HTTP:    &manifest.HTTPConfig{Servers: map[string]manifest.HTTPServerConfig{"main": {}}},
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
		assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/", cfg.RabbiMQ.URI)
		assert.Equal(t, 5000, cfg.GRPC.Servers["main"].Port)
		assert.Equal(t, 8000, cfg.HTTP.Servers["main"].Port)
	})

	t.Run("nil_pointer_in_config/ok", func(t *testing.T) {
		defer resetEnvVars()

		os.Setenv("MONGODB_URI", "mongodb://localhost:27017")

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
