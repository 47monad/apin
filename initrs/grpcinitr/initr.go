package grpcinitr

import (
	"context"

	"github.com/47monad/apin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type ServerShell struct {
	Server       *grpc.Server
	HealthServer *health.Server
}

func MustNew(ctx context.Context, b apin.Builder[*ServerStore]) *ServerShell {
	shell, err := New(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, b apin.Builder[*ServerStore]) (*ServerShell, error) {
	return _init(ctx, b)
}

func _init(ctx context.Context, b apin.Builder[*ServerStore]) (*ServerShell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	shell := &ServerShell{}

	shell.Server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			store.Interceptors...,
		),
	)

	if store.HealthCheck {
		shell.HealthServer = health.NewServer()
		healthgrpc.RegisterHealthServer(shell.Server, shell.HealthServer)
	}

	if store.Runnable != nil {
		store.Runnable(shell.Server)
	}

	if store.Reflection {
		reflection.Register(shell.Server)
	}

	// if store.PromMetrics != nil {
	// 	store.PromMetrics.InitializeMetrics(shell.Server)
	// }

	return shell, nil
}
