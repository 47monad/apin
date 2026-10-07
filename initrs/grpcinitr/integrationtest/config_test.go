package integrationtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/grpcinitr"
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
	Name string           `json:"name" yaml:"name"`
	GRPC grpcinitr.Config `json:"grpc" yaml:"grpc"`
}

func TestConfigFormatsAndEnvironmentOverlay(t *testing.T) {
	t.Run("JSON file, dotenv, and process environment", func(t *testing.T) {
		t.Setenv("API_GRPC_REFLECTION", "false")
		root := t.TempDir()
		configPath := filepath.Join(root, "service.json")
		envPath := filepath.Join(root, ".env")
		if err := os.WriteFile(configPath, []byte(`{"name":"json","grpc":{"servers":{"api":{"port":50051,"reflection":false,"healthCheck":false}},"clients":{"billing":{"target":"dns:///billing:50051"}}}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(envPath, []byte("API_GRPC_REFLECTION=true\nAPI_GRPC_HEALTH_CHECK=true\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		var cfg serviceConfig
		if err := config.Load(configPath, envPath, &cfg); err != nil {
			t.Fatal(err)
		}
		api := cfg.GRPC.Servers["api"]
		if cfg.Name != "json" || api.Port != 50051 || api.Reflection || !api.HealthCheck || cfg.GRPC.Clients["billing"].Target != "dns:///billing:50051" {
			t.Fatalf("unexpected environment overlay: %#v", cfg)
		}
	})

	t.Run("YAML", func(t *testing.T) {
		unsetEnv(t, "API_GRPC_REFLECTION")
		unsetEnv(t, "API_GRPC_HEALTH_CHECK")
		configPath := filepath.Join(t.TempDir(), "service.yml")
		if err := os.WriteFile(configPath, []byte("name: yaml\ngrpc:\n  servers:\n    api:\n      port: 50052\n      reflection: true\n      healthCheck: false\n  clients:\n    billing:\n      target: dns:///billing:50052\n"), 0o600); err != nil {
			t.Fatal(err)
		}

		var cfg serviceConfig
		if err := config.Load(configPath, "", &cfg); err != nil {
			t.Fatal(err)
		}
		api := cfg.GRPC.Servers["api"]
		if cfg.Name != "yaml" || api.Port != 50052 || !api.Reflection || api.HealthCheck || cfg.GRPC.Clients["billing"].Target != "dns:///billing:50052" {
			t.Fatalf("unexpected YAML configuration: %#v", cfg)
		}
	})
}
