package pginitr_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/47monad/apin/initrs/pginitr"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConfigDefaultsAndURI(t *testing.T) {
	stop := errors.New("stop before connect")
	var captured *pgxpool.Config
	_, err := pginitr.New(context.Background(),
		pginitr.WithConfig(&pginitr.Config{Host: "localhost", DBName: "settings"}),
		pginitr.WithNativePoolConfig(func(config *pgxpool.Config) error {
			captured = config
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if captured == nil || captured.ConnConfig.Host != "localhost" || captured.ConnConfig.Database != "settings" {
		t.Fatalf("native config = %+v, want localhost/settings", captured)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  pginitr.Config
		want string
	}{
		{name: "invalid mode", cfg: pginitr.Config{Host: "localhost", Mode: "cluster"}, want: "invalid mode"},
		{name: "invalid ssl mode", cfg: pginitr.Config{Host: "localhost", SSLMode: "yes"}, want: "invalid sslMode"},
		{name: "port out of range", cfg: pginitr.Config{Host: "localhost", Port: 65536}, want: "out of range"},
		{name: "negative timeout", cfg: pginitr.Config{Host: "localhost", ConnTimeout: -1}, want: "connTimeout"},
		{name: "negative pool fields", cfg: pginitr.Config{Host: "localhost", Pool: pginitr.PoolConfig{
			MinConns: -1, MaxConns: -1, MaxConnLifetime: -1, MaxConnIdleTime: -1, HealthCheckInterval: -1,
		}}, want: "must not be negative"},
		{name: "min exceeds max", cfg: pginitr.Config{Host: "localhost", Pool: pginitr.PoolConfig{MinConns: 4, MaxConns: 2}}, want: "must not exceed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pginitr.New(context.Background(), pginitr.WithConfig(&tt.cfg))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("New() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestConfigDSN(t *testing.T) {
	cfg := &pginitr.Config{
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
		cfg := &pginitr.Config{URI: "postgres://uri-host:9999/uridb", Host: "ignored", DBName: "ignored"}
		got, err := cfg.DSN()
		if err != nil {
			t.Fatalf("DSN() error = %v", err)
		}
		if want := "postgres://uri-host:9999/uridb"; got != want {
			t.Errorf("DSN() = %q, want %q", got, want)
		}
	})

	t.Run("default port", func(t *testing.T) {
		cfg := &pginitr.Config{Host: "localhost", DBName: "settings"}
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
		cfg  pginitr.Config
		want string
	}{
		{name: "missing host", cfg: pginitr.Config{DBName: "settings"}, want: "host is required"},
		{name: "missing database", cfg: pginitr.Config{Host: "localhost"}, want: "dbName is required"},
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
	_, err := pginitr.New(context.Background(), pginitr.WithConfig(&pginitr.Config{
		Host: "localhost",
		Pool: pginitr.PoolConfig{
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
