package promgrpcinitr

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// resolvedConfig is private construction state owned by promgrpcinitr.
type resolvedConfig struct {
	registerer prometheus.Registerer
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithRegisterer sets the Prometheus registerer that receives the gRPC server
// metrics. It defaults to prometheus.DefaultRegisterer.
func WithRegisterer(registerer prometheus.Registerer) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.registerer = registerer
		return nil
	})
}

func apply(s *resolvedConfig, opts []Option) error {
	var errs []error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(s); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
