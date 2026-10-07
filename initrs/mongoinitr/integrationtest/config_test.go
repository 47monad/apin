package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin"
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/mongoinitr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ apin.ReadinessChecker = (*mongoinitr.Shell)(nil)

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
	Name  string             `json:"name" yaml:"name"`
	Mongo *mongoinitr.Config `json:"mongo" yaml:"mongo"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file, dotenv, process environment", func(t *testing.T) {
		t.Setenv("MONGODB_URI", "mongodb://process:27017")
		unsetEnv(t, "MONGODB_DB_NAME")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name":"json","mongo":{"uri":"mongodb://file:27017","dbName":"file-db"}}`), 0o600))
		require.NoError(t, os.WriteFile(envPath, []byte("MONGODB_URI=mongodb://dotenv:27017\nMONGODB_DB_NAME=dotenv-db\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, envPath, &cfg))
		require.NotNil(t, cfg.Mongo)
		assert.Equal(t, "json", cfg.Name)
		assert.Equal(t, "mongodb://process:27017", cfg.Mongo.URI)
		assert.Equal(t, "dotenv-db", cfg.Mongo.DBName)
	})

	t.Run("YAML", func(t *testing.T) {
		unsetEnv(t, "MONGODB_URI")
		unsetEnv(t, "MONGODB_DB_NAME")
		configPath := filepath.Join(t.TempDir(), "service.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("name: yaml\nmongo:\n  uri: mongodb://yaml:27017\n  dbName: yaml-db\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, "", &cfg))
		require.NotNil(t, cfg.Mongo)
		assert.Equal(t, "yaml", cfg.Name)
		assert.Equal(t, "mongodb://yaml:27017", cfg.Mongo.URI)
		assert.Equal(t, "yaml-db", cfg.Mongo.DBName)
	})
}
