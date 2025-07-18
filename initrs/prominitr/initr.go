package prominitr

import (
	"context"

	"github.com/47monad/apin"
	"github.com/47monad/zaal"
	"github.com/prometheus/client_golang/prometheus"
)

type Shell struct {
	Registry *prometheus.Registry
}

func MustNewFromConfig(ctx context.Context, config *zaal.PrometheusConfig) *Shell {
	shell, err := NewFromConfig(ctx, config)
	if err != nil {
		panic(err)
	}
	return shell
}

func NewFromConfig(ctx context.Context, config *zaal.PrometheusConfig) (*Shell, error) {
	opts := Opts()
	// TODO: enable grpc metrics here
	return _init(ctx, opts)
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
