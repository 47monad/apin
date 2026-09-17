package grpcinitr

import (
	"context"
	"net"

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

// Serve serves on lis until the context is done, then gracefully stops the
// server and returns. The result is nil after a graceful shutdown, so it can
// be used directly as an apin.Runnable.
func (shell *ServerShell) Serve(ctx context.Context, lis net.Listener) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- shell.Server.Serve(lis)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shell.Server.GracefulStop()
		<-errCh // Serve returns (or ErrServerStopped) after GracefulStop
		return nil
	}
}

// Close gracefully stops the server. Safe to call after Serve already shut
// it down.
func (shell *ServerShell) Close(ctx context.Context) error {
	if shell.Server == nil {
		return nil
	}

	stopped := make(chan struct{})
	go func() {
		shell.Server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		shell.Server.Stop()
		return nil
	}
}
