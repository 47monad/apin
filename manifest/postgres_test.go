package manifest_test

import (
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresDSN(t *testing.T) {
	t.Run("uri_takes_precedence/ok", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{
			URI:     "postgres://uri-host:9999/uridb",
			Host:    "localhost",
			Port:    2231,
			DBName:  "testdb",
			SSLMode: "require",
		}

		dsn, err := cfg.DSN()
		require.NoError(t, err)
		assert.Equal(t, "postgres://uri-host:9999/uridb", dsn)
	})

	t.Run("composed_from_parts/ok", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{
			Host:        "localhost",
			Port:        2231,
			Username:    "postgres",
			Password:    "secret",
			DBName:      "testdb",
			SSLMode:     "require",
			AppName:     "test-app",
			ConnTimeout: 5,
		}

		dsn, err := cfg.DSN()
		require.NoError(t, err)
		assert.Equal(t,
			"postgres://postgres:secret@localhost:2231/testdb?application_name=test-app&connect_timeout=5&sslmode=require",
			dsn)
	})

	t.Run("composed_defaults_port/ok", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{
			Host:   "localhost",
			DBName: "testdb",
		}

		dsn, err := cfg.DSN()
		require.NoError(t, err)
		assert.Equal(t, "postgres://localhost:5432/testdb", dsn)
	})

	t.Run("composed_without_credentials/ok", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{
			Host:   "localhost",
			Port:   5432,
			DBName: "testdb",
		}

		dsn, err := cfg.DSN()
		require.NoError(t, err)
		assert.Equal(t, "postgres://localhost:5432/testdb", dsn)
	})

	t.Run("missing_host/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{DBName: "testdb"}

		_, err := cfg.DSN()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host is required")
	})

	t.Run("missing_dbname/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{Host: "localhost"}

		_, err := cfg.DSN()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dbName is required")
	})
}

func TestPostgresValidate(t *testing.T) {
	t.Run("empty_config/ok", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("valid_config/ok", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{
			Port:        65535,
			Mode:        manifest.PostgresModeSingle,
			SSLMode:     "verify-full",
			ConnTimeout: 5,
			Pool: manifest.PostgresPoolConfig{
				MaxConns:            10,
				MinConns:            2,
				MaxConnLifetime:     300,
				MaxConnIdleTime:     60,
				HealthCheckInterval: 30,
			},
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid_mode/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{Mode: "cluster"}
		assert.ErrorContains(t, cfg.Validate(), "invalid mode")
	})

	t.Run("invalid_ssl_mode/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{SSLMode: "yes"}
		assert.ErrorContains(t, cfg.Validate(), "invalid sslMode")
	})

	t.Run("port_out_of_range/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{Port: 65536}
		assert.ErrorContains(t, cfg.Validate(), "out of range")

		cfg = &manifest.PostgresConfig{Port: -1}
		assert.ErrorContains(t, cfg.Validate(), "out of range")
	})

	t.Run("negative_conn_timeout/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{ConnTimeout: -1}
		assert.ErrorContains(t, cfg.Validate(), "connTimeout")
	})

	t.Run("negative_pool_values/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{Pool: manifest.PostgresPoolConfig{
			MaxConns:            -1,
			MinConns:            -1,
			MaxConnLifetime:     -1,
			MaxConnIdleTime:     -1,
			HealthCheckInterval: -1,
		}}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pool.maxConns")
		assert.Contains(t, err.Error(), "pool.minConns")
		assert.Contains(t, err.Error(), "pool.maxConnLifetime")
		assert.Contains(t, err.Error(), "pool.maxConnIdleTime")
		assert.Contains(t, err.Error(), "pool.healthCheckInterval")
	})

	t.Run("min_conns_exceeds_max_conns/error", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{Pool: manifest.PostgresPoolConfig{
			MaxConns: 2,
			MinConns: 5,
		}}
		assert.ErrorContains(t, cfg.Validate(), "must not exceed")
	})

	t.Run("reports_multiple_errors", func(t *testing.T) {
		cfg := &manifest.PostgresConfig{Mode: "cluster", SSLMode: "yes", Port: 70000}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode")
		assert.Contains(t, err.Error(), "invalid sslMode")
		assert.Contains(t, err.Error(), "out of range")
	})
}
