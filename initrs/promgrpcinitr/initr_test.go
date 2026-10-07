package promgrpcinitr_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/promgrpcinitr"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type testServer interface{}

var testServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.Service",
	HandlerType: (*testServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Unary",
			Handler: func(srv any, ctx context.Context, _ func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				handler := func(context.Context, any) (any, error) { return &emptypb.Empty{}, nil }
				if interceptor == nil {
					return handler(ctx, nil)
				}
				return interceptor(ctx, nil, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/test.Service/Unary"}, handler)
			},
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Stream",
			Handler:       func(any, grpc.ServerStream) error { return nil },
			ServerStreams: true,
		},
	},
}

func grpcTypes(t *testing.T, registry *prometheus.Registry, name string) map[string]bool {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "grpc_type" {
					types[label.GetValue()] = true
				}
			}
		}
	}
	return types
}

func TestUnaryAndStreamMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	shell, err := promgrpcinitr.New(context.Background(), promgrpcinitr.WithRegisterer(registry))
	if err != nil {
		t.Fatal(err)
	}
	if shell.ServerMetrics == nil {
		t.Fatal("New returned nil ServerMetrics")
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(shell.UnaryInterceptor),
		grpc.ChainStreamInterceptor(shell.StreamInterceptor),
	)
	server.RegisterService(&testServiceDesc, struct{}{})
	defer server.Stop()

	listener := bufconn.Listen(1024 * 1024)
	defer func() { _ = listener.Close() }()
	go func() { _ = server.Serve(listener) }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.Invoke(ctx, "/test.Service/Unary", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("unary invoke: %v", err)
	}

	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{StreamName: "Stream", ServerStreams: true}, "/test.Service/Stream")
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatal(err)
	}
	_ = stream.RecvMsg(&emptypb.Empty{})

	types := grpcTypes(t, registry, "grpc_server_handled_total")
	if !types["unary"] {
		t.Fatalf("unary metrics missing: %v", types)
	}
	if !types["server_stream"] {
		t.Fatalf("stream metrics missing: %v", types)
	}
}

func TestNilRegistererRejected(t *testing.T) {
	if _, err := promgrpcinitr.New(context.Background(), promgrpcinitr.WithRegisterer(nil)); err == nil {
		t.Fatal("New() with nil registerer error = nil, want error")
	}
}
