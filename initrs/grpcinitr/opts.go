package grpcinitr

import (
	"errors"

	"google.golang.org/grpc"
)

// Config contains gRPC server features owned by grpcinitr.
type Config struct {
	Reflection  bool `json:"reflection" yaml:"reflection" env:"grpc_reflection"`
	HealthCheck bool `json:"healthCheck" yaml:"healthCheck" env:"grpc_health_check"`
}

// resolvedConfig is private construction state owned by grpcinitr.
type resolvedConfig struct {
	interceptors  []grpc.UnaryServerInterceptor
	serverOptions []grpc.ServerOption
	healthCheck   bool
	reflection    bool
	runnable      func(*grpc.Server)
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
		return apply(s, []Option{
			WithReflection(config.Reflection),
			WithHealthCheck(config.HealthCheck),
		})
	})
}

// WithRunnable registers bootstrap logic to run against the created server,
// such as registering service implementations.
func WithRunnable(runnable func(server *grpc.Server)) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.runnable = runnable
		return nil
	})
}

// WithReflection enables the gRPC reflection service.
func WithReflection(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.reflection = enabled
		return nil
	})
}

// WithHealthCheck registers the standard gRPC health checking service.
func WithHealthCheck(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.healthCheck = enabled
		return nil
	})
}

// WithInterceptor appends a unary interceptor. It is repeatable; each call
// adds another interceptor, applied in the order they are registered.
func WithInterceptor(i grpc.UnaryServerInterceptor) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.interceptors = append(s.interceptors, i)
		return nil
	})
}

// WithServerOptions appends native gRPC server options. It is the escape
// hatch for server capabilities not wrapped by grpcinitr.
func WithServerOptions(options ...grpc.ServerOption) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.serverOptions = append(s.serverOptions, options...)
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
