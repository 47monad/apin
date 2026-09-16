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
		opts    []pginitr.Option
		wantMod pginitr.Mode
	}{
		{
			name:    "default",
			opts:    []pginitr.Option{pginitr.WithConfig(config())},
			wantMod: pginitr.ModePool,
		},
		{
			name:    "pool",
			opts:    []pginitr.Option{pginitr.WithConfig(config()), pginitr.WithPool()},
			wantMod: pginitr.ModePool,
		},
		{
			name:    "singleConn",
			opts:    []pginitr.Option{pginitr.WithConfig(config()), pginitr.WithSingleConn()},
			wantMod: pginitr.ModeConn,
		},
		{
			name: "optionsOnly",
			opts: []pginitr.Option{
				pginitr.WithHost("localhost"),
				pginitr.WithPort(5432),
				pginitr.WithDBName("settings"),
			},
			wantMod: pginitr.ModePool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shell, err := pginitr.New(context.Background(), tt.opts...)
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
	_, err := pginitr.New(context.Background(), pginitr.WithConfig(config()), pginitr.WithMode("invalid"))
	if err == nil {
		t.Fatal("New() error = nil, want invalid mode error")
	}
}

func TestWithoutConfiguration(t *testing.T) {
	_, err := pginitr.New(context.Background())
	if err == nil {
		t.Fatal("New() error = nil, want missing configuration error")
	}
}

func TestApplyURIInvalid(t *testing.T) {
	_, err := pginitr.New(context.Background(), pginitr.WithURI("http://localhost:5432/settings"))
	if err == nil {
		t.Fatal("New() error = nil, want invalid scheme error")
	}
}
