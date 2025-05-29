package prominitr

import (
	"context"

	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"github.com/prometheus/client_golang/prometheus"
)

type Shell struct {
	Registry *prometheus.Registry
}

var Prometheus = initr.AgentFunc[*Store, *Shell](initPrometheus)

func EnsurePrometheus(ctx context.Context, b initropts.Builder[*Store]) *Shell {
	shell, err := initPrometheus(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func initPrometheus(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	_, err := b.Build()
	if err != nil {
		return nil, err
	}

	return &Shell{
		Registry: prometheus.NewRegistry(),
	}, nil
}
