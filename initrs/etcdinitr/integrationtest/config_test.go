package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/etcdinitr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type serviceConfig struct {
	Name string            `json:"name" yaml:"name"`
	Etcd *etcdinitr.Config `json:"etcd" yaml:"etcd"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file, dotenv, process environment", func(t *testing.T) {
		t.Setenv("ETCD_ENDPOINTS", "process:2379")
		t.Setenv("ETCD_USERNAME", "")
		t.Setenv("ETCD_TIMEOUT", "12")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"name":"json","etcd":{"endpoints":"file:2379","username":"file-user","password":"file-pass","timeout":5}}`), 0o600))
		require.NoError(t, os.WriteFile(envPath, []byte("ETCD_ENDPOINTS=dotenv:2379\nETCD_USERNAME=dotenv-user\nETCD_PASSWORD=dotenv-pass\nETCD_TIMEOUT=10\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, envPath, &cfg))
		require.NotNil(t, cfg.Etcd)
		assert.Equal(t, "json", cfg.Name)
		assert.Equal(t, "process:2379", cfg.Etcd.Endpoints)
		assert.Equal(t, "dotenv-user", cfg.Etcd.Username)
		assert.Equal(t, "dotenv-pass", cfg.Etcd.Password)
		require.NotNil(t, cfg.Etcd.Timeout)
		assert.Equal(t, 12, *cfg.Etcd.Timeout)
	})

	t.Run("YAML", func(t *testing.T) {
		t.Setenv("ETCD_ENDPOINTS", "")
		t.Setenv("ETCD_USERNAME", "")
		t.Setenv("ETCD_PASSWORD", "")
		t.Setenv("ETCD_TIMEOUT", "")
		configPath := filepath.Join(t.TempDir(), "service.yaml")
		require.NoError(t, os.WriteFile(configPath, []byte("name: yaml\netcd:\n  endpoints: yaml:2379\n  username: yaml-user\n  password: yaml-pass\n  timeout: 7\n"), 0o600))

		var cfg serviceConfig
		require.NoError(t, config.Load(configPath, "", &cfg))
		require.NotNil(t, cfg.Etcd)
		assert.Equal(t, "yaml", cfg.Name)
		assert.Equal(t, "yaml:2379", cfg.Etcd.Endpoints)
		assert.Equal(t, "yaml-user", cfg.Etcd.Username)
		assert.Equal(t, "yaml-pass", cfg.Etcd.Password)
		require.NotNil(t, cfg.Etcd.Timeout)
		assert.Equal(t, 7, *cfg.Etcd.Timeout)
	})
}
