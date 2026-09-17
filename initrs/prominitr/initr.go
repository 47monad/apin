package prominitr

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type Shell struct {
	Registry *prometheus.Registry
}

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	store := &Store{}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	return &Shell{
		Registry: prometheus.NewRegistry(),
	}, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	return nil
}
