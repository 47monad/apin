package pginitr

import (
	"testing"

	"github.com/47monad/apin/manifest"
)

func TestOptionPrecedenceAndComposition(t *testing.T) {
	store, err := newStore([]Option{
		WithConfig(&manifest.PostgresConfig{
			URI:      "postgres://fileuser:filepass@filehost:5433/filedb?sslmode=require",
			Username: "user",
			Password: "pass",
			Host:     "host",
			Port:     5432,
			DBName:   "db",
		}),
		WithHost("override-host"),
		WithPort(6543),
		WithSSLMode("disable"),
	})
	if err != nil {
		t.Fatalf("newStore() error = %v", err)
	}

	want := "postgres://user:pass@override-host:6543/db?sslmode=disable"
	if got := store.URI.String(); got != want {
		t.Errorf("URI = %q, want %q", got, want)
	}
}

func TestPortOrderIndependence(t *testing.T) {
	store, err := newStore([]Option{WithPort(5432), WithHost("localhost")})
	if err != nil {
		t.Fatalf("newStore() error = %v", err)
	}
	if got, want := store.URI.Host, "localhost:5432"; got != want {
		t.Errorf("URI.Host = %q, want %q", got, want)
	}
}

func TestDefaults(t *testing.T) {
	store, err := newStore([]Option{WithHost("localhost")})
	if err != nil {
		t.Fatalf("newStore() error = %v", err)
	}
	if store.Mode != ModePool {
		t.Errorf("Mode = %q, want default %q", store.Mode, ModePool)
	}
	if got, want := store.URI.Scheme, "postgres"; got != want {
		t.Errorf("URI.Scheme = %q, want %q", got, want)
	}
}

func TestFullConfigMapping(t *testing.T) {
	store, err := newStore([]Option{WithConfig(&manifest.PostgresConfig{
		Host:        "localhost",
		Port:        5432,
		Username:    "postgres",
		Password:    "secret",
		DBName:      "settings",
		SSLMode:     "disable",
		AppName:     "apin",
		ConnTimeout: 5,
		Mode:        "conn",
		Pool: manifest.PostgresPoolConfig{
			MaxConns:        10,
			MinConns:        2,
			MaxConnLifetime: 3600,
			MaxConnIdleTime: 300,
		},
	})})
	if err != nil {
		t.Fatalf("newStore() error = %v", err)
	}

	want := "postgres://postgres:secret@localhost:5432/settings?application_name=apin&connect_timeout=5&sslmode=disable"
	if got := store.URI.String(); got != want {
		t.Errorf("URI = %q, want %q", got, want)
	}
	if store.Mode != ModeConn {
		t.Errorf("Mode = %q, want %q", store.Mode, ModeConn)
	}
	if store.Pool.MaxConns != 10 || store.Pool.MinConns != 2 || store.Pool.MaxConnLifetime != 3600 || store.Pool.MaxConnIdleTime != 300 {
		t.Errorf("Pool = %+v, want mapped pool config", store.Pool)
	}
}
