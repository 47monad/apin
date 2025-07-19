package grpcinitr

import (
	"github.com/47monad/zaal"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"google.golang.org/grpc"
)

type ServerStore struct {
	Interceptors []grpc.UnaryServerInterceptor
	HealthCheck  bool
	Reflection   bool
	PromMetrics  *grpcprom.ServerMetrics
	Runnable     func(*grpc.Server)
}

type ServerBuilder struct {
	Opts []func(*ServerStore) error
}

func (b *ServerBuilder) Build() (*ServerStore, error) {
	store := &ServerStore{}

	for _, opt := range b.Opts {
		if opt == nil {
			continue
		}

		if err := opt(store); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (b *ServerBuilder) WithRunnable(runnable func(store *grpc.Server)) *ServerBuilder {
	b.Opts = append(b.Opts, func(s *ServerStore) error {
		s.Runnable = runnable
		return nil
	})
	return b
}

func (b *ServerBuilder) WithConfig(config *zaal.GRPCServerConfig) *ServerBuilder {
	if config == nil {
		return b
	}
	if config.Features.Reflection {
		b.WithReflection()
	}
	if config.Features.HealthCheck {
		b.WithHealthCheck()
	}
	return b
}

func (b *ServerBuilder) WithHealthCheck() *ServerBuilder {
	b.Opts = append(b.Opts, func(s *ServerStore) error {
		s.HealthCheck = true
		return nil
	})
	return b
}

// func (b *ServerBuilder) SetPrometheus(reg *prometheus.Registry) *ServerBuilder {
// 	b.Opts = append(b.Opts, func(s *ServerStore) error {
// 		promInterceptor, metrics := grpcutil.WithPromMonitoring(reg)
// 		s.PromMetrics = metrics
// 		s.Interceptors = append(s.Interceptors, promInterceptor)
// 		return nil
// 	})
// 	return b
// }

// func (b *ServerBuilder) SetLogging(l logr.Logger) *ServerBuilder {
// 	b.Opts = append(b.Opts, func(b *ServerStore) error {
// 		b.Interceptors = append(b.Interceptors, grpcutil.NewLoggingInterceptor(l))
// 		return nil
// 	})
// 	return b
// }

func (b *ServerBuilder) WithReflection() *ServerBuilder {
	b.Opts = append(b.Opts, func(s *ServerStore) error {
		s.Reflection = true
		return nil
	})
	return b
}

func (b *ServerBuilder) AddInterceptor(i grpc.UnaryServerInterceptor) *ServerBuilder {
	b.Opts = append(b.Opts, func(s *ServerStore) error {
		s.Interceptors = append(s.Interceptors, i)
		return nil
	})
	return b
}

func Opts() *ServerBuilder {
	return &ServerBuilder{}
}
