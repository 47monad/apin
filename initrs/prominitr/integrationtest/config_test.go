package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/prominitr"
)

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
	Name       string            `json:"name" yaml:"name"`
	Prometheus *prominitr.Config `json:"prometheus" yaml:"prometheus"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file then dotenv then process environment", func(t *testing.T) {
		t.Setenv("PROMETHEUS_GRPC_METRICS", "false")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		if err := os.WriteFile(configPath, []byte(`{"name":"json","prometheus":{"grpcMetrics":false}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(envPath, []byte("PROMETHEUS_GRPC_METRICS=true\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		var cfg serviceConfig
		if err := config.Load(configPath, envPath, &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Name != "json" || cfg.Prometheus == nil || cfg.Prometheus.GRPCMetrics {
			t.Fatalf("unexpected process-environment overlay: %#v", cfg)
		}
	})

	t.Run("dotenv overrides YAML file", func(t *testing.T) {
		unsetEnv(t, "PROMETHEUS_GRPC_METRICS")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.yaml")
		envPath := filepath.Join(root, ".env")
		if err := os.WriteFile(configPath, []byte("name: yaml\nprometheus:\n  grpcMetrics: false\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(envPath, []byte("PROMETHEUS_GRPC_METRICS=true\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		var cfg serviceConfig
		if err := config.Load(configPath, envPath, &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Name != "yaml" || cfg.Prometheus == nil || !cfg.Prometheus.GRPCMetrics {
			t.Fatalf("unexpected dotenv overlay: %#v", cfg)
		}
	})
}
