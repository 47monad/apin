package prominitr

import (
	"github.com/47monad/zaal"
)

// Store is the resolved configuration of a shell.
type Store struct {
	GRPCMetrics bool
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*Store) error

// WithConfig applies a zaal config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *zaal.PrometheusConfig) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		// TODO: wire grpcMetrics into grpcinitr
		s.GRPCMetrics = config.GRPCMetrics
		return nil
	}
}

func apply(s *Store, opts []Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(s); err != nil {
			return err
		}
	}
	return nil
}
