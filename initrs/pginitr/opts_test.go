package pginitr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/pginitr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOptionPrecedenceAndCompositionThroughNew(t *testing.T) {
	stop := errors.New("stop before connect")
	var captured *pgx.ConnConfig

	_, err := pginitr.New(context.Background(),
		pginitr.WithConfig(&pginitr.Config{
			URI:         "postgres://fileuser:filepass@filehost:5433/filedb?sslmode=require",
			Username:    "user",
			Password:    "pass",
			Host:        "host",
			Port:        5432,
			DBName:      "db",
			SSLMode:     "require",
			AppName:     "apin",
			ConnTimeout: 5,
		}),
		pginitr.WithHost("override-host"),
		pginitr.WithPort(6543),
		pginitr.WithSSLMode("disable"),
		pginitr.WithSingleConn(),
		pginitr.WithNativeConnConfig(func(config *pgx.ConnConfig) error {
			captured = config.Copy()
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if captured == nil {
		t.Fatal("native config option was not called")
	}
	if captured.Host != "override-host" || captured.Port != 6543 || captured.Database != "db" {
		t.Errorf("native target = %s:%d/%s, want override-host:6543/db", captured.Host, captured.Port, captured.Database)
	}
	if captured.User != "user" || captured.Password != "pass" {
		t.Errorf("native credentials = %q/%q, want user/pass", captured.User, captured.Password)
	}
	if captured.TLSConfig != nil {
		t.Errorf("TLSConfig = %v, want nil for sslmode=disable", captured.TLSConfig)
	}
	if got := captured.RuntimeParams["application_name"]; got != "apin" {
		t.Errorf("application_name = %q, want apin", got)
	}
	if got, want := captured.ConnectTimeout, 5*time.Second; got != want {
		t.Errorf("ConnectTimeout = %v, want %v", got, want)
	}
}

func TestConfigURIPreservesHostWhenDiscreteHostIsUnset(t *testing.T) {
	stop := errors.New("stop before connect")
	var captured *pgx.ConnConfig

	_, err := pginitr.New(context.Background(),
		pginitr.WithConfig(&pginitr.Config{
			URI:  "postgres://uri-host:5432/uridb",
			Mode: pginitr.ModeConn,
		}),
		pginitr.WithNativeConnConfig(func(config *pgx.ConnConfig) error {
			captured = config.Copy()
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if captured == nil {
		t.Fatal("native config option was not called")
	}
	if captured.Host != "uri-host" {
		t.Errorf("native host = %q, want URI host %q", captured.Host, "uri-host")
	}
}

func TestPortOptionOrderIndependenceThroughNew(t *testing.T) {
	stop := errors.New("stop before connect")
	var captured *pgx.ConnConfig

	_, err := pginitr.New(context.Background(),
		pginitr.WithPort(5432),
		pginitr.WithHost("localhost"),
		pginitr.WithDBName("settings"),
		pginitr.WithSingleConn(),
		pginitr.WithNativeConnConfig(func(config *pgx.ConnConfig) error {
			captured = config.Copy()
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if captured == nil || captured.Host != "localhost" || captured.Port != 5432 || captured.Database != "settings" {
		t.Fatalf("native config = %+v, want localhost:5432/settings", captured)
	}
}

func TestPoolConfigAndNativePoolOptionThroughNew(t *testing.T) {
	stop := errors.New("stop before connect")
	var configuredMax int32
	var configuredJitter time.Duration

	_, err := pginitr.New(context.Background(),
		pginitr.WithConfig(&pginitr.Config{
			Host: "localhost",
			Pool: pginitr.PoolConfig{MaxConns: 10, MinConns: 2},
		}),
		pginitr.WithNativePoolConfig(func(config *pgxpool.Config) error {
			configuredMax = config.MaxConns
			config.MaxConnLifetimeJitter = 17 * time.Second
			configuredJitter = config.MaxConnLifetimeJitter
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if configuredMax != 10 {
		t.Errorf("native pool MaxConns = %d, want 10", configuredMax)
	}
	if configuredJitter != 17*time.Second {
		t.Errorf("native MaxConnLifetimeJitter = %v, want %v", configuredJitter, 17*time.Second)
	}
}

func TestLaterConfigResetsEarlierPoolTuning(t *testing.T) {
	stop := errors.New("stop before connect")
	var defaults, got int32

	_, err := pginitr.New(context.Background(),
		pginitr.WithHost("localhost"),
		pginitr.WithNativePoolConfig(func(config *pgxpool.Config) error {
			defaults = config.MaxConns
			return nil
		}),
		pginitr.WithPoolConfig(pginitr.PoolConfig{MaxConns: 10}),
		pginitr.WithConfig(&pginitr.Config{Host: "localhost"}),
		pginitr.WithNativePoolConfig(func(config *pgxpool.Config) error {
			got = config.MaxConns
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if got != defaults {
		t.Errorf("MaxConns after later empty config = %d, want pgx default %d", got, defaults)
	}
}

func TestLaterModeOptionOverridesInvalidEarlierModeBeforeValidation(t *testing.T) {
	stop := errors.New("stop before connect")
	called := false

	_, err := pginitr.New(context.Background(),
		pginitr.WithConfig(&pginitr.Config{Host: "localhost", Mode: "invalid"}),
		pginitr.WithSingleConn(),
		pginitr.WithNativeConnConfig(func(*pgx.ConnConfig) error {
			called = true
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native configuration sentinel", err)
	}
	if !called {
		t.Error("final validation ran before all options were applied")
	}
}
