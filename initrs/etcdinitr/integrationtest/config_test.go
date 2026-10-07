package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin"
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/etcdinitr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ apin.ReadinessChecker = (*etcdinitr.Shell)(nil)

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
	Name string            `json:"name" yaml:"name"`
	Etcd *etcdinitr.Config `json:"etcd" yaml:"etcd"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file, dotenv, process environment", func(t *testing.T) {
		t.Setenv("ETCD_ENDPOINTS", "process-a:2379,process-b:2379")
		unsetEnv(t, "ETCD_USERNAME")
		t.Setenv("ETCD_TIMEOUT", "12")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name":"json","etcd":{"endpoints":["file-a:2379","file-b:2379"],"username":"file-user","password":"file-pass","timeout":5}}`), 0o600))
		require.NoError(t, os.WriteFile(envPath, []byte("ETCD_ENDPOINTS=dotenv-a:2379,dotenv-b:2379\nETCD_USERNAME=dotenv-user\nETCD_PASSWORD=dotenv-pass\nETCD_TIMEOUT=10\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, envPath, &cfg))
		require.NotNil(t, cfg.Etcd)
		assert.Equal(t, "json", cfg.Name)
		assert.Equal(t, []string{"process-a:2379", "process-b:2379"}, cfg.Etcd.Endpoints)
		assert.Equal(t, "dotenv-user", cfg.Etcd.Username)
		assert.Equal(t, "dotenv-pass", cfg.Etcd.Password)
		require.NotNil(t, cfg.Etcd.Timeout)
		assert.Equal(t, 12, *cfg.Etcd.Timeout)
	})

	t.Run("YAML", func(t *testing.T) {
		unsetEnv(t, "ETCD_ENDPOINTS")
		unsetEnv(t, "ETCD_USERNAME")
		unsetEnv(t, "ETCD_PASSWORD")
		unsetEnv(t, "ETCD_TIMEOUT")
		configPath := filepath.Join(t.TempDir(), "service.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("name: yaml\netcd:\n  endpoints:\n    - \"yaml:2379\"\n  username: yaml-user\n  password: yaml-pass\n  timeout: 7\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, "", &cfg))
		require.NotNil(t, cfg.Etcd)
		assert.Equal(t, "yaml", cfg.Name)
		assert.Equal(t, []string{"yaml:2379"}, cfg.Etcd.Endpoints)
		assert.Equal(t, "yaml-user", cfg.Etcd.Username)
		assert.Equal(t, "yaml-pass", cfg.Etcd.Password)
		require.NotNil(t, cfg.Etcd.Timeout)
		assert.Equal(t, 7, *cfg.Etcd.Timeout)
	})
}
