package pginitr

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is the common query surface of *pgx.Conn and *pgxpool.Pool. It lets
// callers run queries without branching on the shell mode.
type Querier interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
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

	poolCloseOnce sync.Once
	poolCloseDone chan struct{}
}

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	config, err := resolveConfig(opts)
	if err != nil {
		return nil, err
	}

	switch config.mode {
	case ModeConn:
		connConfig, err := pgx.ParseConfig(config.uri.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse postgres config: %w", err)
		}
		for _, configure := range config.connOptions {
			if err := configure(connConfig); err != nil {
				return nil, fmt.Errorf("pginitr: native connection config: %w", err)
			}
		}
		conn, err := pgx.ConnectConfig(ctx, connConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to postgres: %w", err)
		}
		return &Shell{
			Mode: ModeConn,
			Conn: conn,
		}, nil
	case ModePool:
		cfg, err := pgxpool.ParseConfig(config.uri.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse postgres config: %w", err)
		}
		if err := applyPoolConfigOptions(cfg, config.poolOptions); err != nil {
			return nil, fmt.Errorf("pginitr: native pool config: %w", err)
		}

		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
		}

		return &Shell{
			Mode:          ModePool,
			Pool:          pool,
			poolCloseDone: make(chan struct{}),
		}, nil
	default:
		return nil, fmt.Errorf("invalid pginitr mode: %q", config.mode)
	}
}

func resolveConfig(opts []Option) (*resolvedConfig, error) {
	config := &resolvedConfig{
		uri:  &url.URL{Scheme: "postgres"},
		mode: ModePool,
	}
	if err := apply(config, opts); err != nil {
		return nil, err
	}

	// Compose the pending port with the host, regardless of option order.
	if config.port != "" {
		if host := config.uri.Hostname(); host != "" {
			config.uri.Host = net.JoinHostPort(host, config.port)
		}
	}

	if config.uri.User == nil && config.uri.Host == "" && config.uri.Path == "" {
		return nil, fmt.Errorf("pginitr: no postgres configuration provided; pass WithConfig, WithURI, or connection options such as WithHost/WithDBName")
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

func applyPoolConfigOptions(config *pgxpool.Config, options []poolConfigOption) error {
	defaults := config.Copy()
	for _, option := range options {
		if option.tuning != nil {
			config.MaxConns = defaults.MaxConns
			config.MinConns = defaults.MinConns
			config.MaxConnLifetime = defaults.MaxConnLifetime
			config.MaxConnIdleTime = defaults.MaxConnIdleTime
			config.HealthCheckPeriod = defaults.HealthCheckPeriod
			applyPoolConfig(config, *option.tuning)
		}
		if option.configure != nil {
			if err := option.configure(config); err != nil {
				return err
			}
		}
	}
	return nil
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

// Ready verifies PostgreSQL connectivity using the shell's active connection
// mode. It satisfies apin.ReadinessChecker. Callers can bound the probe by
// passing a context with a deadline.
func (shell *Shell) Ready(ctx context.Context) error {
	if shell.Conn != nil {
		return shell.Conn.Ping(ctx)
	}
	if shell.Pool != nil {
		return shell.Pool.Ping(ctx)
	}
	return fmt.Errorf("pginitr: shell is not initialized")
}

// Ping is an alias for Ready retained for callers written before the standard
// readiness contract.
func (shell *Shell) Ping(ctx context.Context) error {
	return shell.Ready(ctx)
}

// Close releases the underlying connection or pool.
func (shell *Shell) Close(ctx context.Context) error {
	switch {
	case shell.Conn != nil:
		if err := shell.Conn.Close(ctx); err != nil {
			return fmt.Errorf("failed to close postgres connection: %w", err)
		}
	case shell.Pool != nil:
		shell.poolCloseOnce.Do(func() {
			if shell.poolCloseDone == nil {
				shell.poolCloseDone = make(chan struct{})
			}
			go func() {
				shell.Pool.Close()
				close(shell.poolCloseDone)
			}()
		})
		select {
		case <-shell.poolCloseDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
