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
	configured, err := grpcinitr.New(t.Context(), grpcinitr.WithConfig(&grpcinitr.Config{
		Reflection:  true,
		HealthCheck: true,
	}))
	require.NoError(t, err)
	services := configured.Server.GetServiceInfo()
	require.Contains(t, services, "grpc.health.v1.Health")
	require.Contains(t, services, "grpc.reflection.v1.ServerReflection")

	overridden, err := grpcinitr.New(t.Context(),
		grpcinitr.WithConfig(&grpcinitr.Config{Reflection: true, HealthCheck: true}),
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
