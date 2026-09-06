package pginitr

import (
	"context"
	"fmt"

	"github.com/47monad/apin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Shell struct {
	Pool *pgxpool.Pool
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

	cfg, err := pgxpool.ParseConfig(store.URI.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	return &Shell{
		Pool: pool,
	}, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Pool == nil {
		return nil
	}

	shell.Pool.Close()
	return nil
}
