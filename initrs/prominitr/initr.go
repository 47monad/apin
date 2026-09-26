package prominitr

import (
	"context"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

type Shell struct {
	Registry              *prometheus.Registry
	GRPCServerInterceptor grpc.UnaryServerInterceptor
	GRPCServerMetrics     *grpcprom.ServerMetrics
}

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	store := &Store{}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	shell := &Shell{Registry: prometheus.NewRegistry()}
	if store.GRPCMetrics {
		shell.GRPCServerInterceptor, shell.GRPCServerMetrics = WithPromMonitoring(shell.Registry)
	}
	return shell, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	return nil
}
