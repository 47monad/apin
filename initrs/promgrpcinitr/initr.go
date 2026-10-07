package promgrpcinitr

import (
	"context"
	"errors"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// Shell exposes gRPC server instrumentation backed by a Prometheus registerer.
type Shell struct {
	// UnaryInterceptor records unary RPC metrics. Wire it with
	// grpcinitr.WithInterceptor.
	UnaryInterceptor grpc.UnaryServerInterceptor
	// StreamInterceptor records streaming RPC metrics. Wire it with
	// grpcinitr.WithStreamInterceptor.
	StreamInterceptor grpc.StreamServerInterceptor
	// ServerMetrics is the native metrics handle for advanced use.
	ServerMetrics *grpcprom.ServerMetrics
}

// MustNew is New but panics on failure.
func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

// New registers gRPC server metrics in the configured registerer and returns
// the unary and stream interceptors plus the native metrics handle.
func New(_ context.Context, opts ...Option) (*Shell, error) {
	config := &resolvedConfig{registerer: prometheus.DefaultRegisterer}
	if err := apply(config, opts); err != nil {
		return nil, err
	}
	if config.registerer == nil {
		return nil, errors.New("promgrpcinitr: registerer must not be nil")
	}

	serverMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	config.registerer.MustRegister(serverMetrics)

	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}

	return &Shell{
		UnaryInterceptor:  serverMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
		StreamInterceptor: serverMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)),
		ServerMetrics:     serverMetrics,
	}, nil
}

// Close is a no-op: the shell holds no external resources.
func (shell *Shell) Close(context.Context) error {
	return nil
}
