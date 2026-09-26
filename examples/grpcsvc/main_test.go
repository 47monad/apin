package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/47monad/apin/config"
)

func TestExampleAggregateLoadsSelectedInitializers(t *testing.T) {
	var cfg serviceConfig
	if err := config.Load("config.json", "", &cfg); err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.HTTP.Servers["api"].Port != 4747 {
		t.Errorf("HTTP.Servers[api].Port = %d, want 4747", cfg.HTTP.Servers["api"].Port)
	}
	if !cfg.GRPC.Reflection || !cfg.GRPC.HealthCheck {
		t.Errorf("GRPC = %+v, want reflection and health check enabled", cfg.GRPC)
	}
	if cfg.Postgres == nil || cfg.Postgres.Host != "localhost" {
		t.Errorf("Postgres = %+v, want localhost configuration", cfg.Postgres)
	}
}

func TestHTTPConfigFileDotEnvAndProcessPrecedence(t *testing.T) {
	t.Setenv("API_HTTP_PORT", "4911")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "service.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  servers:\n    api:\n      port: 4910\n"), 0o600); err != nil {
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
}
