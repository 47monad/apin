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
// distinguishes "unset" from "set to empty", so a default assertion would
// be affected by ambient variables. Tests
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

	// Optional sections stay nil when the instance omits them.
	assert.Nil(t, cfg.HTTP)
}

// TestSchemaHTTPPortConstraint guards the `common` import used by the HTTP
// interface schema: if it fails to resolve, the constraint silently stops
// being enforced instead of erroring out at load time.
func TestSchemaHTTPPortConstraint(t *testing.T) {
	t.Run("valid ports are accepted", func(t *testing.T) {
		cfg := mustBuildInstance(t, `{
			name: "ports"
			http: {servers: {main: {port: 8080}}}
		}`)
		assert.Equal(t, 8080, cfg.HTTP.Servers["main"].Port)
	})

	t.Run("default ports are applied", func(t *testing.T) {
		cfg := mustBuildInstance(t, `{
			name: "ports"
			http: {servers: {main: {}}}
		}`)
		assert.Equal(t, 4747, cfg.HTTP.Servers["main"].Port)
	})

	for _, tc := range []struct {
		name    string
		service string
	}{
		{"zero http port", `{name: "ports", http: {servers: {main: {port: 0}}}}`},
		{"out of range http port", `{name: "ports", http: {servers: {main: {port: 70_000}}}}`},
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
