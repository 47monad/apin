package prominitr

import (
	"context"

	"github.com/47monad/apin"
	"github.com/prometheus/client_golang/prometheus"
)

type Shell struct {
	Registry *prometheus.Registry
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
	_, err := b.Build()
	if err != nil {
		return nil, err
	}

	return &Shell{
		Registry: prometheus.NewRegistry(),
	}, nil
}
