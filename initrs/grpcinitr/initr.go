package grpcinitr

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type ServerShell struct {
	Server       *grpc.Server
	HealthServer *health.Server
}

func MustNew(ctx context.Context, opts ...Option) *ServerShell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*ServerShell, error) {
	store := &ServerStore{}
	if err := apply(store, opts); err != nil {
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

	return shell, nil
}
