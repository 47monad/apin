package prominitr

// Config contains Prometheus settings owned by prominitr.
type Config struct {
	GRPCMetrics bool `json:"grpcMetrics" yaml:"grpcMetrics" env:"prometheus_grpc_metrics"`
}

// Store is the resolved configuration of a shell.
type Store struct {
	GRPCMetrics bool
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*Store) error

// WithConfig applies an initializer-owned config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *Config) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		s.GRPCMetrics = config.GRPCMetrics
		return nil
	}
}

// WithGRPCMetrics enables or disables exposing gRPC server metrics.
func WithGRPCMetrics(enabled bool) Option {
	return func(s *Store) error {
		s.GRPCMetrics = enabled
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
