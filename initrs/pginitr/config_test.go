package pginitr

import (
	"context"
	"strings"
	"testing"
)

func TestConfigDefaultsAndURI(t *testing.T) {
	store, err := newStore([]Option{WithConfig(&Config{
		Host:   "localhost",
		DBName: "settings",
	})})
	if err != nil {
		t.Fatalf("newStore() error = %v", err)
	}
	if store.Mode != ModePool {
		t.Errorf("Mode = %q, want %q", store.Mode, ModePool)
	}
	if got, want := store.URI.String(), "postgres://localhost/settings"; got != want {
		t.Errorf("URI = %q, want %q", got, want)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{name: "invalid mode", cfg: Config{Host: "localhost", Mode: "cluster"}, want: "invalid mode"},
		{name: "invalid ssl mode", cfg: Config{Host: "localhost", SSLMode: "yes"}, want: "invalid sslMode"},
		{name: "port out of range", cfg: Config{Host: "localhost", Port: 65536}, want: "out of range"},
		{name: "negative timeout", cfg: Config{Host: "localhost", ConnTimeout: -1}, want: "connTimeout"},
		{name: "negative pool fields", cfg: Config{Host: "localhost", Pool: PoolConfig{
			MinConns: -1, MaxConns: -1, MaxConnLifetime: -1, MaxConnIdleTime: -1, HealthCheckInterval: -1,
		}}, want: "must not be negative"},
		{name: "min exceeds max", cfg: Config{Host: "localhost", Pool: PoolConfig{MinConns: 4, MaxConns: 2}}, want: "must not exceed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(context.Background(), WithConfig(&tt.cfg))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("newStore() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestConfigDSN(t *testing.T) {
	cfg := &Config{
		Host:        "localhost",
		Port:        2231,
		Username:    "postgres",
		Password:    "secret",
		DBName:      "testdb",
		SSLMode:     "require",
		AppName:     "test-app",
		ConnTimeout: 5,
	}
	got, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN() error = %v", err)
	}
	if want := "postgres://postgres:secret@localhost:2231/testdb?application_name=test-app&connect_timeout=5&sslmode=require"; got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}

func TestConfigDSNURIAndDefaults(t *testing.T) {
	t.Run("URI takes precedence", func(t *testing.T) {
		cfg := &Config{URI: "postgres://uri-host:9999/uridb", Host: "ignored", DBName: "ignored"}
		got, err := cfg.DSN()
		if err != nil {
			t.Fatalf("DSN() error = %v", err)
		}
		if want := "postgres://uri-host:9999/uridb"; got != want {
			t.Errorf("DSN() = %q, want %q", got, want)
		}
	})

	t.Run("default port", func(t *testing.T) {
		cfg := &Config{Host: "localhost", DBName: "settings"}
		got, err := cfg.DSN()
		if err != nil {
			t.Fatalf("DSN() error = %v", err)
		}
		if want := "postgres://localhost:5432/settings"; got != want {
			t.Errorf("DSN() = %q, want %q", got, want)
		}
	})

	for _, tt := range []struct {
		name string
		cfg  Config
		want string
	}{
		{name: "missing host", cfg: Config{DBName: "settings"}, want: "host is required"},
		{name: "missing database", cfg: Config{Host: "localhost"}, want: "dbName is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.cfg.DSN()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("DSN() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestConfigValidationErrorOrder(t *testing.T) {
	_, err := New(context.Background(), WithConfig(&Config{
		Host: "localhost",
		Pool: PoolConfig{
			MaxConnLifetime:     -1,
			MaxConnIdleTime:     -1,
			HealthCheckInterval: -1,
		},
	}))
	if err == nil {
		t.Fatal("New() error = nil, want pool validation errors")
	}
	want := "postgres: pool.maxConnLifetime must not be negative\n" +
		"postgres: pool.maxConnIdleTime must not be negative\n" +
		"postgres: pool.healthCheckInterval must not be negative"
	if err.Error() != want {
		t.Errorf("New() error = %q, want %q", err, want)
	}
}
