package manifest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildWith writes a manifest instance and an optional .env file to a temp dir
// and builds them. Callers that want to observe only the .env file and their
// own variables must call clearEnv first; nothing here reads or writes the
// process environment.
func buildWith(t *testing.T, service, envFile string) (*manifest.Config, error) {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "service.cue")
	if err := os.WriteFile(configPath, []byte("service: "+service+"\n"), 0o600); err != nil {
		t.Fatalf("write instance: %v", err)
	}

	envPath := filepath.Join(dir, "nonexistent.env")
	if envFile != "" {
		envPath = filepath.Join(dir, ".env")
		if err := os.WriteFile(envPath, []byte(envFile), 0o600); err != nil {
			t.Fatalf("write env file: %v", err)
		}
	}
	return manifest.Build(configPath, envPath)
}

func TestParseEnvFile(t *testing.T) {
	t.Run("nonexistent_env/error", func(t *testing.T) {
		_, err := manifest.ParseEnvFile("nonexistent.env")
		assert.Error(t, err)
	})

	t.Run("values are parsed without touching the process env", func(t *testing.T) {
		clearEnv(t)
		path := filepath.Join(t.TempDir(), ".env")
		require.NoError(t, os.WriteFile(path, []byte("LOG_LEVEL=warn\nHOST=from-file\n"), 0o600))

		vars, err := manifest.ParseEnvFile(path)
		require.NoError(t, err)

		value, ok := vars.Lookup("LOG_LEVEL")
		assert.True(t, ok)
		assert.Equal(t, "warn", value)
		_, ok = vars.Lookup("NOT_IN_FILE")
		assert.False(t, ok, "Lookup must not invent variables")

		assert.Empty(t, os.Getenv("LOG_LEVEL"), "parsing must not mutate the process env")
		assert.Empty(t, os.Getenv("HOST"), "parsing must not mutate the process env")
	})
}

func TestLookupPrecedence(t *testing.T) {
	clearEnv(t)
	vars := manifest.EnvVars{"LOG_LEVEL": "from-file", "ONLY_FILE": "file", "EMPTY": ""}

	t.Run("file value is used when the process env has none", func(t *testing.T) {
		value, ok := vars.Lookup("ONLY_FILE")
		assert.True(t, ok)
		assert.Equal(t, "file", value)
	})

	t.Run("process env wins over the file", func(t *testing.T) {
		t.Setenv("LOG_LEVEL", "from-process")
		value, ok := vars.Lookup("LOG_LEVEL")
		assert.True(t, ok)
		assert.Equal(t, "from-process", value)
	})

	t.Run("empty values count as unset", func(t *testing.T) {
		// godotenv would have written "" into the environment, and the
		// overlay has always ignored empty values, so the file's empty
		// value must not shadow the default either.
		_, ok := vars.Lookup("EMPTY")
		assert.False(t, ok)
	})
}

// TestBuildDoesNotMutateProcessEnv is the regression test for the .env file
// being loaded into the global environment: the values must reach the returned
// Config and nothing else.
func TestBuildDoesNotMutateProcessEnv(t *testing.T) {
	clearEnv(t)
	before := os.Environ()

	cfg, err := buildWith(t,
		`{name: "isolation"}`,
		"LOG_LEVEL=warn\nHOST=from-file\n",
	)
	require.NoError(t, err)

	// The file is applied to the config...
	assert.Equal(t, "warn", cfg.Logging.Level)
	assert.Equal(t, "from-file", cfg.Host)

	// ...and to nothing else.
	assert.Equal(t, before, os.Environ(), "Build must not change the process environment")
}

func TestBuildEnvPrecedence(t *testing.T) {
	t.Run("process env overrides the env file", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("LOG_LEVEL", "from-process")
		cfg, err := buildWith(t, `{name: "prec"}`, "LOG_LEVEL=from-file\n")
		require.NoError(t, err)
		assert.Equal(t, "from-process", cfg.Logging.Level)
	})

	t.Run("env file overrides the manifest", func(t *testing.T) {
		clearEnv(t)
		cfg, err := buildWith(t, `{name: "prec", logging: {level: "info"}}`, "LOG_LEVEL=fatal\n")
		require.NoError(t, err)
		assert.Equal(t, "fatal", cfg.Logging.Level)
	})

	t.Run("manifest is used when neither sets a value", func(t *testing.T) {
		clearEnv(t)
		cfg, err := buildWith(t, `{name: "prec", logging: {level: "info"}}`, "")
		require.NoError(t, err)
		assert.Equal(t, "info", cfg.Logging.Level)
	})

	t.Run("postgres section is allocated from the env file alone", func(t *testing.T) {
		// The section is missing from the manifest; only a POSTGRES_*
		// variable brings it back, and the file has to count.
		clearEnv(t)
		cfg, err := buildWith(t, `{name: "prec"}`, "POSTGRES_HOST=db-from-file\n")
		require.NoError(t, err)
		require.NotNil(t, cfg.Postgres)
		assert.Equal(t, "db-from-file", cfg.Postgres.Host)
	})
}

// TestBuildConcurrent covers the other half of the issue: with the env file
// parsed into an isolated set, concurrent builds no longer share variables
// through the process environment. Every goroutine asserts it got its own
// file's value.
func TestBuildConcurrent(t *testing.T) {
	clearEnv(t)

	// Levels valid per the schema's #Level, reused across iterations so a
	// build cannot accidentally read another goroutine's value.
	levels := []string{"debug", "info", "warn", "fatal"}
	const iterations = 20
	goroutines := len(levels) * 2

	dir := t.TempDir()
	configPath := filepath.Join(dir, "service.cue")
	require.NoError(t, os.WriteFile(configPath, []byte("service: {name: \"concurrent\"}\n"), 0o600))

	envPaths := make([]string, len(levels))
	for i, level := range levels {
		envPath := filepath.Join(dir, fmt.Sprintf("level-%d.env", i))
		require.NoError(t, os.WriteFile(envPath, []byte("LOG_LEVEL="+level+"\n"), 0o600))
		envPaths[i] = envPath
	}

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*iterations)
	start := make(chan struct{})

	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start // maximize overlap
			for range iterations {
				i := g % len(levels)
				cfg, err := manifest.Build(configPath, envPaths[i])
				if err != nil {
					errs <- fmt.Errorf("goroutine %d: %w", g, err)
					return
				}
				if cfg.Logging.Level != levels[i] {
					errs <- fmt.Errorf("goroutine %d: Logging.Level = %q, want %q from its own env file",
						g, cfg.Logging.Level, levels[i])
					return
				}
			}
		}(g)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}

	// The whole run must not have leaked a single variable.
	for _, level := range levels {
		if v := os.Getenv("LOG_LEVEL"); v != "" {
			t.Errorf("LOG_LEVEL = %q, want the process environment untouched (wanted level %s)", v, level)
		}
	}
}

func TestOverlayNilConfig(t *testing.T) {
	err := manifest.EnvVars{"LOG_LEVEL": "warn"}.Overlay(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestBuildErrorWrapping(t *testing.T) {
	// A bad env file must still surface as a wrapped error rather than being
	// swallowed, now that it is parsed instead of loaded.
	clearEnv(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "service.cue")
	require.NoError(t, os.WriteFile(configPath, []byte("service: {name: \"x\"}\n"), 0o600))

	envPath := filepath.Join(dir, ".env")
	// Unterminated quote: godotenv fails to parse it.
	require.NoError(t, os.WriteFile(envPath, []byte("LOG_LEVEL=\"unterminated\n"), 0o600))

	_, err := manifest.Build(configPath, envPath)
	require.Error(t, err)
	assert.True(t,
		strings.Contains(err.Error(), "load env file"),
		"error %v should say the env file failed to load", err)
}
