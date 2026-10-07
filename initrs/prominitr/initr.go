package prominitr

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Shell holds the service's private Prometheus registry.
type Shell struct {
	Registry *prometheus.Registry
}

// MustNew is New but panics on failure.
func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

// New constructs a fresh registry, optionally registering the standard Go and
// process collectors. gRPC instrumentation lives in promgrpcinitr.
func New(_ context.Context, opts ...Option) (*Shell, error) {
	config := &resolvedConfig{}
	if err := apply(config, opts); err != nil {
		return nil, err
	}

	registry := prometheus.NewRegistry()
	if config.goCollector {
		registry.MustRegister(collectors.NewGoCollector())
	}
	if config.processCollector {
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	return &Shell{Registry: registry}, nil
}

// Close is a no-op: the shell holds no external resources.
func (shell *Shell) Close(context.Context) error {
	return nil
}
