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

const defaultPort = 50051

type ServerShell struct {
	Server       *grpc.Server
	HealthServer *health.Server
	Port         int
}

// ClientShell owns one outgoing gRPC connection and exposes the native handle.
type ClientShell struct {
	Conn *grpc.ClientConn
}

func MustNew(ctx context.Context, opts ...Option) *ServerShell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*ServerShell, error) {
	config := &resolvedConfig{}
	if err := apply(config, opts); err != nil {
		return nil, err
	}
	return newServer(ctx, config)
}

// NewServer constructs one independently managed server from an entry in
// Config.Servers.
func NewServer(ctx context.Context, config ServerConfig, opts ...Option) (*ServerShell, error) {
	resolved := &resolvedConfig{}
	if err := apply(resolved, []Option{WithConfig(&config)}); err != nil {
		return nil, err
	}
	if err := apply(resolved, opts); err != nil {
		return nil, err
	}
	return newServer(ctx, resolved)
}

// NewClient constructs one independently managed outgoing connection from an
// entry in Config.Clients. grpc.NewClient is lazy; it does not guarantee the
// remote endpoint is reachable when this function returns.
func NewClient(_ context.Context, config ClientConfig, opts ...ClientOption) (*ClientShell, error) {
	resolved := &clientConfig{target: config.Target}
	if err := applyClient(resolved, opts); err != nil {
		return nil, err
	}
	if resolved.target == "" {
		return nil, errors.New("grpcinitr: client target is required")
	}
	conn, err := grpc.NewClient(resolved.target, resolved.dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("grpcinitr: create client %q: %w", resolved.target, err)
	}
	return &ClientShell{Conn: conn}, nil
}

// MustNewServer is NewServer but panics on failure.
func MustNewServer(ctx context.Context, config ServerConfig, opts ...Option) *ServerShell {
	shell, err := NewServer(ctx, config, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func newServer(_ context.Context, config *resolvedConfig) (*ServerShell, error) {
	if config.port == 0 {
		config.port = defaultPort
	}
	if config.port < 1 || config.port > 65535 {
		return nil, fmt.Errorf("grpcinitr: port must be between 1 and 65535, got %d", config.port)
	}

	shell := &ServerShell{Port: config.port}

	serverOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			config.interceptors...,
		),
	}
	serverOptions = append(serverOptions, config.serverOptions...)
	shell.Server = grpc.NewServer(serverOptions...)

	if config.healthCheck {
		shell.HealthServer = health.NewServer()
		healthgrpc.RegisterHealthServer(shell.Server, shell.HealthServer)
	}

	if config.runnable != nil {
		config.runnable(shell.Server)
	}

	if config.reflection {
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
	if shell == nil || shell.Server == nil {
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

// Close closes the native client connection. gRPC's ClientConn.Close is
// immediate and does not take a context; ctx is accepted to implement
// apin.Closer consistently with other shells.
func (shell *ClientShell) Close(context.Context) error {
	if shell == nil || shell.Conn == nil {
		return nil
	}
	return shell.Conn.Close()
}
