package manifest_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZaal(t *testing.T) {
	res, err := manifest.Build(
		"./testdata/main.cue",
		"./testdata/main.env",
	)
	if err != nil {
		t.Fatal(err)
	}

	js, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(js))
}

func TestPostgresCUESchema(t *testing.T) {
	// Build loads env files into the process environment, which leaks
	// across tests (issue #31). Clear postgres vars so fixtures get a
	// clean slate; they are unset for the process, not restored.
	pgEnvVars := []string{
		"POSTGRES_URI", "POSTGRES_HOST", "POSTGRES_PORT",
		"POSTGRES_USERNAME", "POSTGRES_PASSWORD", "POSTGRES_DB_NAME",
		"POSTGRES_SSL_MODE", "POSTGRES_APP_NAME", "POSTGRES_CONN_TIMEOUT",
		"POSTGRES_MODE", "POSTGRES_POOL_MAX_CONNS", "POSTGRES_POOL_MIN_CONNS",
		"POSTGRES_POOL_MAX_CONN_LIFETIME", "POSTGRES_POOL_MAX_CONN_IDLE_TIME",
		"POSTGRES_POOL_HEALTH_CHECK_INTERVAL",
	}
	for _, name := range pgEnvVars {
		os.Unsetenv(name)
	}

	t.Run("valid_config/ok", func(t *testing.T) {
		cfg, err := manifest.Build("./testdata/postgres/main.cue", "nonexistent.env")
		require.NoError(t, err)
		require.NotNil(t, cfg.Postgres)

		pg := cfg.Postgres
		assert.Equal(t, "localhost", pg.Host)
		assert.Equal(t, 65535, pg.Port)
		assert.Equal(t, "testdb", pg.DBName)
		assert.Equal(t, "require", pg.SSLMode)
		assert.Equal(t, 5, pg.ConnTimeout)
		assert.Equal(t, "single", pg.Mode)
		assert.Equal(t, 10, pg.Pool.MaxConns)
		assert.Equal(t, 2, pg.Pool.MinConns)
		assert.Equal(t, 300, pg.Pool.MaxConnLifetime)
		assert.Equal(t, 60, pg.Pool.MaxConnIdleTime)
		assert.Equal(t, 30, pg.Pool.HealthCheckInterval)
	})

	t.Run("mode_defaults_to_pool/ok", func(t *testing.T) {
		cfg, err := manifest.Build("./testdata/main.cue", "./testdata/main.env")
		require.NoError(t, err)
		require.NotNil(t, cfg.Postgres)
		assert.Equal(t, "pool", cfg.Postgres.Mode)
	})

	t.Run("bad_port/error", func(t *testing.T) {
		_, err := manifest.Build("./testdata/postgres_bad_port/main.cue", "nonexistent.env")
		assert.Error(t, err)
	})

	t.Run("float_port/error", func(t *testing.T) {
		_, err := manifest.Build("./testdata/postgres_float_port/main.cue", "nonexistent.env")
		assert.Error(t, err)
	})

	t.Run("bad_mode/error", func(t *testing.T) {
		_, err := manifest.Build("./testdata/postgres_bad_mode/main.cue", "nonexistent.env")
		assert.Error(t, err)
	})

	t.Run("bad_sslmode/error", func(t *testing.T) {
		_, err := manifest.Build("./testdata/postgres_bad_sslmode/main.cue", "nonexistent.env")
		assert.Error(t, err)
	})

	t.Run("bad_pool_conns/error", func(t *testing.T) {
		_, err := manifest.Build("./testdata/postgres_bad_pool/main.cue", "nonexistent.env")
		assert.Error(t, err)
	})
}

func TestGRPCClientAddressDefault(t *testing.T) {
	// Build loads env files into the process environment. Clear client
	// address vars so the empty CUE default is what we observe.
	for _, name := range []string{"UWCL_GRPC_CLIENT_ADDRESS", "GRPC_CLIENT_ADDRESS"} {
		os.Unsetenv(name)
	}

	t.Run("empty_client_uses_default_address/ok", func(t *testing.T) {
		cfg, err := manifest.Build("./testdata/grpc_client_default/main.cue", "nonexistent.env")
		require.NoError(t, err)
		require.NotNil(t, cfg.GRPC)
		client, ok := cfg.GRPC.Clients["uwcl"]
		require.True(t, ok)
		assert.Equal(t, "", client.Address)
	})

	t.Run("env_overrides_default_address/ok", func(t *testing.T) {
		t.Setenv("UWCL_GRPC_CLIENT_ADDRESS", "localhost:50051")
		cfg, err := manifest.Build("./testdata/grpc_client_default/main.cue", "nonexistent.env")
		require.NoError(t, err)
		require.NotNil(t, cfg.GRPC)
		assert.Equal(t, "localhost:50051", cfg.GRPC.Clients["uwcl"].Address)
	})
}
