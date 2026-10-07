package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/config"
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

func TestExampleAggregateLoadsSelectedInitializers(t *testing.T) {
	for _, path := range []string{"config.json", "config.yaml"} {
		t.Run(path, func(t *testing.T) {
			unsetEnv(t, "API_HTTP_PORT")
			unsetEnv(t, "API_GRPC_REFLECTION")
			unsetEnv(t, "API_GRPC_HEALTH_CHECK")
			unsetEnv(t, "POSTGRES_HOST")

			cfg, err := loadServiceConfig(path)
			if err != nil {
				t.Fatalf("loadServiceConfig() error = %v", err)
			}
			if cfg.HTTP.Servers["api"].Port != 4747 {
				t.Errorf("HTTP.Servers[api].Port = %d, want 4747", cfg.HTTP.Servers["api"].Port)
			}
			if !cfg.GRPC.Servers["api"].Reflection || !cfg.GRPC.Servers["api"].HealthCheck {
				t.Errorf("GRPC.Servers[api] = %+v, want reflection and health check enabled", cfg.GRPC.Servers["api"])
			}
			if cfg.GRPC.Servers["admin"].Port != 50052 || cfg.GRPC.Clients["billing"].Target != "dns:///billing:50051" {
				t.Errorf("GRPC aggregate missing named resources: %+v", cfg.GRPC)
			}
			if cfg.Postgres == nil || cfg.Postgres.Host != "localhost" {
				t.Errorf("Postgres = %+v, want localhost configuration", cfg.Postgres)
			}
		})
	}
}

func TestHTTPConfigFileDotEnvAndProcessPrecedence(t *testing.T) {
	t.Setenv("API_HTTP_PORT", "4911")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "service.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  servers:\n    api:\n      port: 4910\ngrpc:\n  servers:\n    api:\n      port: 50051\n      reflection: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("API_HTTP_PORT=4912\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var cfg serviceConfig
	if err := config.Load(configPath, envPath, &cfg); err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.HTTP.Servers["api"].Port != 4911 {
		t.Errorf("HTTP.Servers[api].Port = %d, want process environment value 4911", cfg.HTTP.Servers["api"].Port)
	}
	t.Setenv("API_GRPC_REFLECTION", "true")
	if err := config.Load(configPath, "", &cfg); err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if !cfg.GRPC.Servers["api"].Reflection {
		t.Error("GRPC.Servers[api].Reflection = false, want process environment overlay true")
	}
}
