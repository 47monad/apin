package grpcinitr

import (
	"errors"

	"google.golang.org/grpc"
)

type resolvedConfig struct {
	port               int
	interceptors       []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor
	serverOptions      []grpc.ServerOption
	healthCheck        bool
	reflection         bool
	runnable           func(*grpc.Server)
}

// Option is a sealed functional option accepted by New and NewServer.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error { return option(config) }

// WithConfig applies configuration for one server. For multiple named
// resources, select an entry from Config.Servers and call NewServer separately.
func WithConfig(config *ServerConfig) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if config == nil {
			return nil
		}
		s.port = config.Port
		s.reflection = config.Reflection
		s.healthCheck = config.HealthCheck
		return nil
	})
}

// WithPort selects the server's TCP port; zero selects the default port.
func WithPort(port int) Option {
	return optionFunc(func(s *resolvedConfig) error { s.port = port; return nil })
}

// WithRunnable registers bootstrap logic such as service implementations.
func WithRunnable(runnable func(server *grpc.Server)) Option {
	return optionFunc(func(s *resolvedConfig) error { s.runnable = runnable; return nil })
}

// WithReflection enables the gRPC reflection service.
func WithReflection(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error { s.reflection = enabled; return nil })
}

// WithHealthCheck registers the standard gRPC health checking service.
func WithHealthCheck(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error { s.healthCheck = enabled; return nil })
}

// WithInterceptor appends a unary interceptor in registration order.
func WithInterceptor(i grpc.UnaryServerInterceptor) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.interceptors = append(s.interceptors, i)
		return nil
	})
}

// WithStreamInterceptor appends a stream interceptor in registration order.
func WithStreamInterceptor(i grpc.StreamServerInterceptor) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.streamInterceptors = append(s.streamInterceptors, i)
		return nil
	})
}

// WithServerOptions appends native gRPC server options.
func WithServerOptions(options ...grpc.ServerOption) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.serverOptions = append(s.serverOptions, options...)
		return nil
	})
}

type clientConfig struct {
	target      string
	dialOptions []grpc.DialOption
}

// ClientOption is a sealed functional option for native gRPC client settings.
type ClientOption interface {
	apply(*clientConfig) error
}

type clientOptionFunc func(*clientConfig) error

func (option clientOptionFunc) apply(config *clientConfig) error { return option(config) }

// WithDialOptions passes native gRPC dial options through to grpc.NewClient.
func WithDialOptions(options ...grpc.DialOption) ClientOption {
	return clientOptionFunc(func(config *clientConfig) error {
		config.dialOptions = append(config.dialOptions, options...)
		return nil
	})
}

func apply(config *resolvedConfig, options []Option) error {
	var errs []error
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option.apply(config); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func applyClient(config *clientConfig, options []ClientOption) error {
	var errs []error
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option.apply(config); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
