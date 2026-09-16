package pginitr

import (
	"context"
	"fmt"
	"time"

	"github.com/47monad/apin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Shell struct {
	// Mode reports which connection strategy the shell was initialized with.
	Mode Mode
	// Pool is set when Mode is ModePool.
	Pool *pgxpool.Pool
	// Conn is set when Mode is ModeConn.
	Conn *pgx.Conn
}

func MustNew(ctx context.Context, b apin.Builder[*Store]) *Shell {
	shell, err := _init(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, b apin.Builder[*Store]) (*Shell, error) {
	return _init(ctx, b)
}

func _init(ctx context.Context, b apin.Builder[*Store]) (*Shell, error) {
	store, err := b.Build()
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
