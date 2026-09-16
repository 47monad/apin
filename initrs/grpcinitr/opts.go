package grpcinitr

import (
	"github.com/47monad/zaal"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"google.golang.org/grpc"
)

// ServerStore is the resolved configuration of a server shell.
type ServerStore struct {
	Interceptors []grpc.UnaryServerInterceptor
	HealthCheck  bool
	Reflection   bool
	PromMetrics  *grpcprom.ServerMetrics
	Runnable     func(*grpc.Server)
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*ServerStore) error

// WithConfig applies a zaal config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *zaal.GRPCServerConfig) Option {
	return func(s *ServerStore) error {
		if config == nil {
			return nil
		}
		return apply(s, []Option{
			WithReflection(config.Features.Reflection),
			WithHealthCheck(config.Features.HealthCheck),
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

// AddInterceptor appends a unary interceptor.
func AddInterceptor(i grpc.UnaryServerInterceptor) Option {
	return func(s *ServerStore) error {
		s.Interceptors = append(s.Interceptors, i)
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
