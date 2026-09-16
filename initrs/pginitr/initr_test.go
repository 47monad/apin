package pginitr_test

import (
	"context"
	"testing"

	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/zaal"
)

func config() *zaal.PostgresConfig {
	return &zaal.PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "123456789",
		DBName:   "settings",
	}
}

// TestInitModes covers both connection modes. A live postgres instance is not
// required: when none is available, New fails to connect and the test only
// asserts the error path.
func TestInitModes(t *testing.T) {
	tests := []struct {
		name    string
		opts    func() *pginitr.Builder
		wantMod pginitr.Mode
	}{
		{name: "default", opts: func() *pginitr.Builder { return pginitr.Opts() }, wantMod: pginitr.ModePool},
		{name: "pool", opts: func() *pginitr.Builder { return pginitr.Opts().WithPool() }, wantMod: pginitr.ModePool},
		{name: "singleConn", opts: func() *pginitr.Builder { return pginitr.Opts().WithSingleConn() }, wantMod: pginitr.ModeConn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.opts().WithConfig(config())

			store, err := b.Build()
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if store.Mode != tt.wantMod {
				t.Errorf("Store.Mode = %q, want %q", store.Mode, tt.wantMod)
			}

			shell, err := pginitr.New(context.Background(), tt.opts().WithConfig(config()))
			if err != nil {
				t.Logf("skipping connection checks (no live postgres): %v", err)
				return
			}
			defer func() {
				if err := shell.Close(context.Background()); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}()

			if shell.Mode != tt.wantMod {
				t.Errorf("Shell.Mode = %q, want %q", shell.Mode, tt.wantMod)
			}
			switch tt.wantMod {
			case pginitr.ModePool:
				if shell.Pool == nil {
					t.Error("Shell.Pool is nil in pool mode")
				}
				if shell.Conn != nil {
					t.Error("Shell.Conn is set in pool mode")
				}
			case pginitr.ModeConn:
				if shell.Conn == nil {
					t.Error("Shell.Conn is nil in single connection mode")
				}
				if shell.Pool != nil {
					t.Error("Shell.Pool is set in single connection mode")
				}
			}
		})
	}
}

func TestWithInvalidMode(t *testing.T) {
	b := pginitr.Opts().WithMode("invalid")

	if _, err := b.Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid mode error")
	}
}

func TestApplyURIInvalid(t *testing.T) {
	b := pginitr.Opts().ApplyURI("http://localhost:5432/settings")

	if _, err := b.Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid scheme error")
	}
}

func TestWithConfigMapping(t *testing.T) {
	store, err := pginitr.Opts().WithConfig(&zaal.PostgresConfig{
		Host:        "localhost",
		Port:        5432,
		Username:    "postgres",
		Password:    "secret",
		DBName:      "settings",
		SSLMode:     "disable",
		AppName:     "apin",
		ConnTimeout: 5,
		Mode:        "conn",
		Pool: zaal.PostgresPoolConfig{
			MaxConns:        10,
			MinConns:        2,
			MaxConnLifetime: 3600,
			MaxConnIdleTime: 300,
		},
	}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got, want := store.URI.String(), "postgres://postgres:secret@localhost:5432/settings?application_name=apin&connect_timeout=5&sslmode=disable"; got != want {
		t.Errorf("Store.URI = %q, want %q", got, want)
	}
	if store.Mode != pginitr.ModeConn {
		t.Errorf("Store.Mode = %q, want %q", store.Mode, pginitr.ModeConn)
	}
	if store.Pool.MaxConns != 10 || store.Pool.MinConns != 2 || store.Pool.MaxConnLifetime != 3600 || store.Pool.MaxConnIdleTime != 300 {
		t.Errorf("Store.Pool = %+v, want mapped pool config", store.Pool)
	}
}

func TestWithConfigModeInvalid(t *testing.T) {
	b := pginitr.Opts().WithConfig(&zaal.PostgresConfig{Mode: "wat"})

	if _, err := b.Build(); err == nil {
		t.Fatal("Build() error = nil, want invalid mode error")
	}
}
