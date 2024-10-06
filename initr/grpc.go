package initr

import (
	"context"
	"github.com/47monad/apin/initropts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type GrpcServerShell struct {
	Server       *grpc.Server
	HealthServer *health.Server
}

var GrpcServer = AgentFunc[*initropts.GrpcServerStore, *GrpcServerShell](initGrpcServer)

func initGrpcServer(ctx context.Context, b initropts.Builder[*initropts.GrpcServerStore]) (*GrpcServerShell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	shell := &GrpcServerShell{}

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

	if store.PromMetrics != nil {
		store.PromMetrics.InitializeMetrics(shell.Server)
	}

	return shell, nil
}
