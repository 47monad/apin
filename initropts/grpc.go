package initropts

import (
	"github.com/47monad/apin/internal/grpcutil"
	"github.com/47monad/apin/internal/logger"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

type GrpcServerStore struct {
	Interceptors []grpc.UnaryServerInterceptor
	HealthCheck  bool
	Reflection   bool
	PromMetrics  *grpcprom.ServerMetrics
	Runnable     func(*grpc.Server)
}

type GrpcServerBuilder struct {
	Opts []func(*GrpcServerStore) error
}

func (b *GrpcServerBuilder) Build() (*GrpcServerStore, error) {
	store := &GrpcServerStore{}

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

func (b *GrpcServerBuilder) WithRunnable(runnable func(store *grpc.Server)) *GrpcServerBuilder {
	b.Opts = append(b.Opts, func(s *GrpcServerStore) error {
		s.Runnable = runnable
		return nil
	})
	return b
}

func (b *GrpcServerBuilder) WithHealthCheck() *GrpcServerBuilder {
	b.Opts = append(b.Opts, func(s *GrpcServerStore) error {
		s.HealthCheck = true
		return nil
	})
	return b
}

func (b *GrpcServerBuilder) SetPrometheus(reg *prometheus.Registry) *GrpcServerBuilder {
	b.Opts = append(b.Opts, func(s *GrpcServerStore) error {
		promInterceptor, metrics := grpcutil.WithPromMonitoring(reg)
		s.PromMetrics = metrics
		s.Interceptors = append(s.Interceptors, promInterceptor)
		return nil
	})
	return b
}

func (b *GrpcServerBuilder) SetLogging(l logger.Logger) *GrpcServerBuilder {
	b.Opts = append(b.Opts, func(b *GrpcServerStore) error {
		b.Interceptors = append(b.Interceptors, grpcutil.NewLoggingInterceptor(l))
		return nil
	})
	return b
}
func (b *GrpcServerBuilder) WithReflection() *GrpcServerBuilder {
	b.Opts = append(b.Opts, func(s *GrpcServerStore) error {
		s.Reflection = true
		return nil
	})
	return b
}

func (b *GrpcServerBuilder) AddInterceptor(i grpc.UnaryServerInterceptor) *GrpcServerBuilder {
	b.Opts = append(b.Opts, func(s *GrpcServerStore) error {
		s.Interceptors = append(s.Interceptors, i)
		return nil
	})
	return b
}

func GrpcServer() *GrpcServerBuilder {
	return &GrpcServerBuilder{}
}
