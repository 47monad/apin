package integrationtest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/47monad/apin"
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/redisinitr"
)

var _ apin.ReadinessChecker = (*redisinitr.Shell)(nil)

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
	Redis *redisinitr.Config `json:"redis" yaml:"redis"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file, dotenv, process environment", func(t *testing.T) {
		t.Setenv("REDIS_ADDRESSES", "process-a:6379,process-b:6379")
		t.Setenv("REDIS_DATABASE", "4")
		unsetEnv(t, "REDIS_USERNAME")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		if err := os.WriteFile(configPath, []byte(`{"name":"json","redis":{"addresses":["file-a:6379"],"username":"file-user","password":"file-pass","database":1}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(envPath, []byte("REDIS_ADDRESSES=dotenv-a:6379,dotenv-b:6379\nREDIS_USERNAME=dotenv-user\nREDIS_PASSWORD=dotenv-pass\nREDIS_DATABASE=2\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		var cfg serviceConfig
		if err := config.Load(configPath, envPath, &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Name != "json" {
			t.Errorf("Name = %q, want json", cfg.Name)
		}
		if cfg.Redis == nil {
			t.Fatal("Redis config is nil")
		}
		if want := []string{"process-a:6379", "process-b:6379"}; !reflect.DeepEqual(cfg.Redis.Addresses, want) {
			t.Errorf("Addresses = %v, want %v", cfg.Redis.Addresses, want)
		}
		if cfg.Redis.Username != "dotenv-user" || cfg.Redis.Password != "dotenv-pass" {
			t.Errorf("credentials = %q/%q, want dotenv-user/dotenv-pass", cfg.Redis.Username, cfg.Redis.Password)
		}
		if cfg.Redis.Database != 4 {
			t.Errorf("Database = %d, want 4", cfg.Redis.Database)
		}
	})

	t.Run("YAML", func(t *testing.T) {
		for _, key := range []string{"REDIS_ADDRESSES", "REDIS_USERNAME", "REDIS_PASSWORD", "REDIS_DATABASE", "REDIS_MASTER_NAME"} {
			unsetEnv(t, key)
		}
		configPath := filepath.Join(t.TempDir(), "service.yaml")
		if err := os.WriteFile(configPath, []byte("name: yaml\nredis:\n  addresses:\n    - \"yaml-a:6379\"\n    - \"yaml-b:6379\"\n  username: yaml-user\n  database: 2\n  masterName: yaml-master\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		var cfg serviceConfig
		if err := config.Load(configPath, "", &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Redis == nil {
			t.Fatal("Redis config is nil")
		}
		if want := []string{"yaml-a:6379", "yaml-b:6379"}; !reflect.DeepEqual(cfg.Redis.Addresses, want) {
			t.Errorf("Addresses = %v, want %v", cfg.Redis.Addresses, want)
		}
		if cfg.Redis.Username != "yaml-user" || cfg.Redis.Database != 2 || cfg.Redis.MasterName != "yaml-master" {
			t.Errorf("unexpected YAML config: %#v", cfg.Redis)
		}
	})
}
