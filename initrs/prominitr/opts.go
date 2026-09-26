package prominitr

import "errors"

// Config contains Prometheus settings owned by prominitr.
type Config struct {
	GRPCMetrics bool `json:"grpcMetrics" yaml:"grpcMetrics" env:"prometheus_grpc_metrics"`
}

// resolvedConfig is private construction state owned by prominitr.
type resolvedConfig struct {
	grpcMetrics bool
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithConfig applies an initializer-owned config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *Config) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if config == nil {
			return nil
		}
		s.grpcMetrics = config.GRPCMetrics
		return nil
	})
}

// WithGRPCMetrics enables or disables exposing gRPC server metrics.
func WithGRPCMetrics(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.grpcMetrics = enabled
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
