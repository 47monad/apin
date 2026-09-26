package grpcinitr

import (
	"google.golang.org/grpc"
)

// Config contains gRPC server features owned by grpcinitr.
type Config struct {
	Reflection  bool `json:"reflection" yaml:"reflection" env:"grpc_reflection"`
	HealthCheck bool `json:"healthCheck" yaml:"healthCheck" env:"grpc_health_check"`
}

// ServerStore is the resolved configuration of a server shell.
type ServerStore struct {
	Interceptors  []grpc.UnaryServerInterceptor
	ServerOptions []grpc.ServerOption
	HealthCheck   bool
	Reflection    bool
	Runnable      func(*grpc.Server)
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*ServerStore) error

// WithConfig applies an initializer-owned config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *Config) Option {
	return func(s *ServerStore) error {
		if config == nil {
			return nil
		}
		return apply(s, []Option{
			WithReflection(config.Reflection),
			WithHealthCheck(config.HealthCheck),
		})
	}
}

// WithRunnable registers bootstrap logic to run against the created server,
// such as registering service implementations.
func WithRunnable(runnable func(server *grpc.Server)) Option {
	return func(s *ServerStore) error {
		s.Runnable = runnable
		return nil
	}
}

// WithReflection enables the gRPC reflection service.
func WithReflection(enabled bool) Option {
	return func(s *ServerStore) error {
		s.Reflection = enabled
		return nil
	}
}

// WithHealthCheck registers the standard gRPC health checking service.
func WithHealthCheck(enabled bool) Option {
	return func(s *ServerStore) error {
		s.HealthCheck = enabled
		return nil
	}
}

// WithInterceptor appends a unary interceptor. It is repeatable; each call
// adds another interceptor, applied in the order they are registered.
func WithInterceptor(i grpc.UnaryServerInterceptor) Option {
	return func(s *ServerStore) error {
		s.Interceptors = append(s.Interceptors, i)
		return nil
	}
}

// WithServerOptions appends native gRPC server options. It is the escape
// hatch for server capabilities not wrapped by grpcinitr.
func WithServerOptions(options ...grpc.ServerOption) Option {
	return func(s *ServerStore) error {
		s.ServerOptions = append(s.ServerOptions, options...)
		return nil
	}
}

func apply(s *ServerStore, opts []Option) error {
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
