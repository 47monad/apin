package grpcinitr

import (
	"context"
	"errors"
	"fmt"
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

	serverOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			store.Interceptors...,
		),
	}
	serverOptions = append(serverOptions, store.ServerOptions...)
	shell.Server = grpc.NewServer(serverOptions...)

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

// Serve serves on lis until the context is done or the native server stops.
// It returns on context cancellation; the caller's lifecycle owner then calls
// Close with its shutdown context so graceful stopping uses the same deadline
// as the other tracked shells.
func (shell *ServerShell) Serve(ctx context.Context, lis net.Listener) error {
	if shell == nil || shell.Server == nil {
		return errors.New("grpcinitr: cannot serve with a nil server")
	}
	if lis == nil {
		return errors.New("grpcinitr: cannot serve with a nil listener")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- shell.Server.Serve(lis)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return err
	case <-ctx.Done():
		// The caller's lifecycle owner closes the shell with its single
		// shutdown context and deadline.
		return nil
	}
}

// Close gracefully stops the server. Safe to call after Serve already shut
// it down.
func (shell *ServerShell) Close(ctx context.Context) error {
	if shell.Server == nil {
		return nil
	}
	if shell.HealthServer != nil {
		shell.HealthServer.Shutdown()
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
		return fmt.Errorf("grpcinitr: server did not drain in time: %w", ctx.Err())
	}
}
