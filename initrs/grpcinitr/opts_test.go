package grpcinitr_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestConfigAndOptionsControlFeatures(t *testing.T) {
	configured, err := grpcinitr.New(t.Context(), grpcinitr.WithConfig(&grpcinitr.ServerConfig{
		Reflection:  true,
		HealthCheck: true,
	}))
	require.NoError(t, err)
	services := configured.Server.GetServiceInfo()
	require.Contains(t, services, "grpc.health.v1.Health")
	require.Contains(t, services, "grpc.reflection.v1.ServerReflection")

	overridden, err := grpcinitr.New(t.Context(),
		grpcinitr.WithConfig(&grpcinitr.ServerConfig{Reflection: true, HealthCheck: true}),
		grpcinitr.WithReflection(false),
		grpcinitr.WithHealthCheck(false),
	)
	require.NoError(t, err)
	services = overridden.Server.GetServiceInfo()
	require.NotContains(t, services, "grpc.health.v1.Health")
	require.NotContains(t, services, "grpc.reflection.v1.ServerReflection")

	optionOnly, err := grpcinitr.New(t.Context(), grpcinitr.WithReflection(true))
	require.NoError(t, err)
	require.Contains(t, optionOnly.Server.GetServiceInfo(), "grpc.reflection.v1.ServerReflection")
}

func TestNamedConfigCreatesIndependentlyManagedServersAndClients(t *testing.T) {
	config := grpcinitr.Config{
		Servers: map[string]grpcinitr.ServerConfig{
			"public": {Reflection: true, Port: 50051},
			"admin":  {HealthCheck: true, Port: 50052},
		},
		Clients: map[string]grpcinitr.ClientConfig{
			"billing": {Target: "passthrough:///billing"},
			"orders":  {Target: "passthrough:///orders"},
		},
	}

	public, err := grpcinitr.NewServer(t.Context(), config.Servers["public"])
	require.NoError(t, err)
	admin, err := grpcinitr.NewServer(t.Context(), config.Servers["admin"])
	require.NoError(t, err)
	billing, err := grpcinitr.NewClient(t.Context(), config.Clients["billing"],
		grpcinitr.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	require.NoError(t, err)
	orders, err := grpcinitr.NewClient(t.Context(), config.Clients["orders"],
		grpcinitr.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	require.NoError(t, err)

	require.NotSame(t, public, admin)
	require.Contains(t, public.Server.GetServiceInfo(), "grpc.reflection.v1.ServerReflection")
	require.NotContains(t, public.Server.GetServiceInfo(), "grpc.health.v1.Health")
	require.Contains(t, admin.Server.GetServiceInfo(), "grpc.health.v1.Health")
	require.NotContains(t, admin.Server.GetServiceInfo(), "grpc.reflection.v1.ServerReflection")
	require.Equal(t, 50051, public.Port)
	require.Equal(t, 50052, admin.Port)
	require.NotNil(t, billing.Conn)
	require.NotNil(t, orders.Conn)

	require.NoError(t, public.Close(t.Context()))
	require.NoError(t, billing.Close(t.Context()))
	require.Equal(t, "SHUTDOWN", billing.Conn.GetState().String())
	require.NotEqual(t, "SHUTDOWN", orders.Conn.GetState().String())
	require.NoError(t, admin.Close(t.Context()))
	require.NoError(t, orders.Close(t.Context()))
	require.Equal(t, "SHUTDOWN", orders.Conn.GetState().String())
}

func TestNamedServerConfigValidation(t *testing.T) {
	_, err := grpcinitr.NewServer(t.Context(), grpcinitr.ServerConfig{Port: 65536})
	require.ErrorContains(t, err, "grpcinitr: port must be between 1 and 65535")
	_, err = grpcinitr.NewClient(t.Context(), grpcinitr.ClientConfig{})
	require.ErrorContains(t, err, "grpcinitr: client target is required")
}

func TestServerOptionsAndInterceptorsRemainAvailable(t *testing.T) {
	interceptorCalled := false
	nativeOption := grpc.UnknownServiceHandler(func(_ any, _ grpc.ServerStream) error {
		return status.Error(codes.Unimplemented, "native server option reached")
	})
	shell, err := grpcinitr.New(t.Context(),
		grpcinitr.WithServerOptions(nativeOption),
		grpcinitr.WithHealthCheck(true),
		grpcinitr.WithInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			interceptorCalled = true
			return handler(ctx, req)
		}),
	)
	require.NoError(t, err)
	defer shell.Close(t.Context())

	listener := bufconn.Listen(1024 * 1024)
	defer listener.Close()
	go func() { _ = shell.Server.Serve(listener) }()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	_, err = grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	require.True(t, interceptorCalled)

	err = conn.Invoke(ctx, "/unknown.Service/Method", &emptypb.Empty{}, &emptypb.Empty{})
	require.Error(t, err)
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.True(t, strings.Contains(err.Error(), "native server option reached"))
}

func TestRunHealthCheckUpdatesHealthServiceAndStopsOnContext(t *testing.T) {
	shell, err := grpcinitr.New(t.Context(), grpcinitr.WithHealthCheck(true))
	require.NoError(t, err)
	defer shell.Close(t.Context())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	checked := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- shell.RunHealthCheck(ctx, "api", time.Millisecond, func(context.Context) bool {
			select {
			case checked <- struct{}{}:
			default:
			}
			return false
		})
	}()

	select {
	case <-checked:
	case <-time.After(time.Second):
		t.Fatal("health checker was not called")
	}
	response, err := shell.HealthServer.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{Service: "api"})
	require.NoError(t, err)
	require.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, response.Status)

	cancel()
	require.NoError(t, <-done)
}

func TestServeReturnsOnContextAndCloseStopsNativeServer(t *testing.T) {
	shell, err := grpcinitr.New(t.Context(), grpcinitr.WithHealthCheck(true))
	require.NoError(t, err)

	listener := bufconn.Listen(1024 * 1024)
	ctx, cancel := context.WithCancel(t.Context())
	serveDone := make(chan error, 1)
	go func() { serveDone <- shell.Serve(ctx, listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()
	checkCtx, checkCancel := context.WithTimeout(t.Context(), time.Second)
	defer checkCancel()
	_, err = grpc_health_v1.NewHealthClient(conn).Check(checkCtx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)

	cancel()
	require.NoError(t, <-serveDone)

	closeCtx, closeCancel := context.WithTimeout(t.Context(), time.Second)
	defer closeCancel()
	require.NoError(t, shell.Close(closeCtx))
	require.NoError(t, listener.Close())
}

func TestCloseUsesCallerDeadlineToForceStop(t *testing.T) {
	started := make(chan struct{})
	shell, err := grpcinitr.New(t.Context(), grpcinitr.WithServerOptions(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
		close(started)
		<-stream.Context().Done()
		return stream.Context().Err()
	})))
	require.NoError(t, err)

	listener := bufconn.Listen(1024 * 1024)
	serveDone := make(chan error, 1)
	go func() { serveDone <- shell.Serve(t.Context(), listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	invokeDone := make(chan error, 1)
	go func() {
		invokeDone <- conn.Invoke(context.Background(), "/unknown.Service/Method", &emptypb.Empty{}, &emptypb.Empty{})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not reach the blocking handler")
	}

	closeCtx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	err = shell.Close(closeCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, <-serveDone)
	require.NoError(t, listener.Close())
	select {
	case <-invokeDone:
	case <-time.After(time.Second):
		t.Fatal("forced stop did not unblock the active request")
	}
}
