package pginitr

import (
	"context"
	"fmt"

	"github.com/47monad/apin"
	"github.com/jackc/pgx/v5"
)

type Shell struct {
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

	conn, err := pgx.Connect(ctx, store.URI.String())
	if err != nil {
		panic(err)
	}

	return &Shell{
		Conn: conn,
	}, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Conn == nil {
		return nil
	}

	err := shell.Conn.Close(ctx)
	if err != nil {
		return fmt.Errorf("failed to close postgres connection: %v", err)
	}
	return nil
}
