package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clearEnv empties the process environment for the duration of the test and
// restores it afterwards.
//
// t.Setenv cannot be used here: it can only set a value, and the overlay
// distinguishes "unset" from "set to empty" — an empty POSTGRES_* variable
// still counts as present when the optional postgres section is allocated, so
// a test asserting that the section stays nil would see it allocated. Tests
// that need a variable *set* use t.Setenv, which restores itself; this helper
// is for the ones that need a variable *absent*.
//
// Build no longer loads env files into the process environment (issue #41),
// but a test asserting the schema defaults must still be shielded from the
// developer's own environment. Tests in a package run sequentially, so
// nothing else observes the empty environment.
func clearEnv(t *testing.T) {
	t.Helper()
	saved := os.Environ()
	for _, entry := range saved {
		name, _, _ := strings.Cut(entry, "=")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
	}
	t.Cleanup(func() {
		for _, entry := range saved {
			name, value, _ := strings.Cut(entry, "=")
			os.Setenv(name, value)
		}
	})
}

// buildInstance writes a manifest instance to a temp dir and builds it against
// the embedded schema, with a clean environment so only the schema's own
// defaults apply. The env file does not exist.
func buildInstance(t *testing.T, service string) (*manifest.Config, error) {
	t.Helper()
	clearEnv(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "service.cue")
	if err := os.WriteFile(path, []byte("service: "+service+"\n"), 0o600); err != nil {
		t.Fatalf("write instance: %v", err)
	}
	return manifest.Build(path, filepath.Join(dir, "nonexistent.env"))
}

func mustBuildInstance(t *testing.T, service string) *manifest.Config {
	t.Helper()
	cfg, err := buildInstance(t, service)
	require.NoError(t, err)
	return cfg
}

// TestSchemaDefaults pins the defaults the embedded CUE schema applies to an
// instance that leaves sections out. Every sibling package defines
// `#Section` and exports `section: #Section`, so these defaults are the only
// place the schema's behavior is pinned down.
func TestSchemaDefaults(t *testing.T) {
	cfg := mustBuildInstance(t, `{name: "defaults-probe"}`)

	assert.Equal(t, "defaults-probe", cfg.Name)
	assert.Equal(t, "Go App", cfg.Title)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.Equal(t, "127.0.0.1", cfg.Host)
	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "normal", cfg.Mode)

	// The logging section is omitted entirely, so the log package's default
	// level must still materialize on decode.
	assert.Equal(t, "error", cfg.Logging.Level)

	// Optional sections stay nil when the instance omits them.
	assert.Nil(t, cfg.Mongodb)
	assert.Nil(t, cfg.Postgres)
	assert.Nil(t, cfg.Etcd)
	assert.Nil(t, cfg.RabbitMQ)
	assert.Nil(t, cfg.Prometheus)
	assert.Nil(t, cfg.GRPC)
	assert.Nil(t, cfg.HTTP)
}

func TestSchemaLoggingLevels(t *testing.T) {
	t.Run("explicit level wins", func(t *testing.T) {
		cfg := mustBuildInstance(t, `{name: "lvl", logging: {level: "debug"}}`)
		assert.Equal(t, "debug", cfg.Logging.Level)
	})

	t.Run("empty section falls back to the default", func(t *testing.T) {
		cfg := mustBuildInstance(t, `{name: "lvl", logging: {}}`)
		assert.Equal(t, "error", cfg.Logging.Level)
	})

	t.Run("level outside #Level is rejected", func(t *testing.T) {
		_, err := buildInstance(t, `{name: "lvl", logging: {level: "verbose"}}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "logging.level")
	})
}

// TestSchemaCommonPortConstraint guards the `common` import used by the
// interface and db packages: if those imports fail to resolve, the constraint
// silently stops being enforced instead of erroring out at load time.
func TestSchemaCommonPortConstraint(t *testing.T) {
	t.Run("valid ports are accepted", func(t *testing.T) {
		cfg := mustBuildInstance(t, `{
			name: "ports"
			http: {servers: {main: {port: 8080}}}
			grpc: {servers: {main: {port: 50051}}}
		}`)
		assert.Equal(t, 8080, cfg.HTTP.Servers["main"].Port)
		assert.Equal(t, 50051, cfg.GRPC.Servers["main"].Port)
	})

	t.Run("default ports are applied", func(t *testing.T) {
		cfg := mustBuildInstance(t, `{
			name: "ports"
			http: {servers: {main: {}}}
			grpc: {servers: {main: {}}}
		}`)
		assert.Equal(t, 4747, cfg.HTTP.Servers["main"].Port)
		assert.Equal(t, 4748, cfg.GRPC.Servers["main"].Port)
	})

	for _, tc := range []struct {
		name    string
		service string
	}{
		{"zero http port", `{name: "ports", http: {servers: {main: {port: 0}}}}`},
		{"out of range http port", `{name: "ports", http: {servers: {main: {port: 70_000}}}}`},
		{"zero grpc port", `{name: "ports", grpc: {servers: {main: {port: 0}}}}`},
		{"out of range grpc port", `{name: "ports", grpc: {servers: {main: {port: 70_000}}}}`},
	} {
		t.Run(tc.name+" is rejected", func(t *testing.T) {
			_, err := buildInstance(t, tc.service)
			require.Error(t, err)
			// The error has to point at the port field, i.e. come from the
			// schema constraint rather than something unrelated.
			assert.Contains(t, err.Error(), ".port")
		})
	}
}
