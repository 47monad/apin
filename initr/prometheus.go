package initr

import (
	"context"
	"github.com/47monad/apin/initropts"
	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusShell struct {
	Registry *prometheus.Registry
}

var Prometheus = AgentFunc[*initropts.PrometheusStore, *PrometheusShell](initPrometheus)

func EnsurePrometheus(ctx context.Context, b initropts.Builder[*initropts.PrometheusStore]) *PrometheusShell {
	shell, err := initPrometheus(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func initPrometheus(ctx context.Context, b initropts.Builder[*initropts.PrometheusStore]) (*PrometheusShell, error) {
	_, err := b.Build()
	if err != nil {
		return nil, err
	}

	return &PrometheusShell{
		Registry: prometheus.NewRegistry(),
	}, nil
}
