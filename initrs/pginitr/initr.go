package pginitr

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the common query surface of *pgx.Conn and *pgxpool.Pool. It lets
// callers run queries without branching on the shell mode.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

var (
	_ Querier = (*pgx.Conn)(nil)
	_ Querier = (*pgxpool.Pool)(nil)
)

// Shell holds the postgres connection. Exactly one of Pool or Conn is
// non-nil, selected by Mode.
type Shell struct {
	// Mode reports which connection strategy the shell was initialized with.
	Mode Mode
	// Pool is set when Mode is ModePool.
	Pool *pgxpool.Pool
	// Conn is set when Mode is ModeConn.
	Conn *pgx.Conn
}

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	store, err := newStore(opts)
	if err != nil {
		return nil, err
	}

	switch store.Mode {
	case ModeConn:
		conn, err := pgx.Connect(ctx, store.URI.String())
		if err != nil {
			return nil, fmt.Errorf("failed to connect to postgres: %w", err)
		}
		return &Shell{
			Mode: ModeConn,
			Conn: conn,
		}, nil
	case ModePool:
		cfg, err := pgxpool.ParseConfig(store.URI.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse postgres config: %w", err)
		}
		applyPoolConfig(cfg, store.Pool)

		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
		}

		return &Shell{
			Mode: ModePool,
			Pool: pool,
		}, nil
	default:
		return nil, fmt.Errorf("invalid pginitr mode: %q", store.Mode)
	}
}

func newStore(opts []Option) (*Store, error) {
	store := &Store{
		URI:  &url.URL{Scheme: "postgres"},
		Mode: ModePool,
	}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	// Compose the pending port with the host, regardless of option order.
	if store.Port != "" {
		if host := store.URI.Hostname(); host != "" {
			store.URI.Host = net.JoinHostPort(host, store.Port)
		}
	}

	if store.URI.User == nil && store.URI.Host == "" && store.URI.Path == "" {
		return nil, fmt.Errorf("pginitr: no postgres configuration provided; pass WithConfig, WithURI, or connection options such as WithHost/WithDBName")
	}

	return store, nil
}

func applyPoolConfig(cfg *pgxpool.Config, pool PoolConfig) {
	if pool.MaxConns > 0 {
		cfg.MaxConns = int32(pool.MaxConns)
	}
	if pool.MinConns > 0 {
		cfg.MinConns = int32(pool.MinConns)
	}
	if pool.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = time.Duration(pool.MaxConnLifetime) * time.Second
	}
	if pool.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = time.Duration(pool.MaxConnIdleTime) * time.Second
	}
	if pool.HealthCheckInterval > 0 {
		cfg.HealthCheckPeriod = time.Duration(pool.HealthCheckInterval) * time.Second
	}
}

// DB returns the query surface of the shell, regardless of mode. Mode-specific
// capabilities (e.g. Listen on Conn, Acquire/Stat on Pool) are available
// through the exported fields.
func (shell *Shell) DB() (Querier, error) {
	if shell.Conn != nil {
		return shell.Conn, nil
	}
	if shell.Pool != nil {
		return shell.Pool, nil
	}
	return nil, fmt.Errorf("pginitr: shell is not initialized")
}

// Close releases the underlying connection or pool.
func (shell *Shell) Close(ctx context.Context) error {
	switch {
	case shell.Conn != nil:
		if err := shell.Conn.Close(ctx); err != nil {
			return fmt.Errorf("failed to close postgres connection: %w", err)
		}
	case shell.Pool != nil:
		shell.Pool.Close()
	}
	return nil
}
