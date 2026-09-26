# grpcinitr

gRPC server initr. Returns a shell holding a configured
[`grpc.Server`](https://pkg.go.dev/google.golang.org/grpc#Server), with
optional health checking and reflection, ready for registration and serving.

```bash
go get github.com/47monad/apin/initrs/grpcinitr
```

## Usage

```go
// app-owned config; the initializer owns only server features
srvShell, err := grpcinitr.New(ctx, grpcinitr.WithConfig(&grpcinitr.Config{
	Reflection: true,
	HealthCheck: true,
}))

// config file + service registration + interceptors
srvShell, err := grpcinitr.New(ctx,
	grpcinitr.WithConfig(&grpcCfg.Features),
	grpcinitr.WithRunnable(func(s *grpc.Server) {
		pb.RegisterUserServiceServer(s, &userServer{db: dbShell})
	}),
	grpcinitr.WithInterceptor(authInterceptor),
	grpcinitr.WithServerOptions(grpc.MaxRecvMsgSize(4<<20)),
)
```

## Serving

`Serve` is context-aware and directly usable as an `apin.App` runnable. When
the app cancels runnable contexts, `Serve` returns; App then closes tracked
shells with its single shutdown deadline, and the shell gracefully stops its
native gRPC server:

```go
lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcCfg.Port))

app.Run(ctx, func(ctx context.Context) error {
	return srvShell.Serve(ctx, lis)
})
```

When the standard health service is enabled, `RunHealthCheck(ctx, service,
interval, checker)` periodically updates its status and returns when `ctx` is
cancelled. The checker should honor its context.

## Shell

```go
type ServerShell struct {
	Server       *grpc.Server
	HealthServer *health.Server // set when health check is enabled
}
```

## Options

| Option | Description |
|---|---|
| `WithConfig(*grpcinitr.Config)` | apply initializer-owned feature settings |
| `WithReflection(enabled bool)` | register the gRPC reflection service |
| `WithHealthCheck(enabled bool)` | register the standard gRPC health checking service |
| `WithRunnable(fn func(*grpc.Server))` | bootstrap logic run against the created server — where services get registered |
| `WithInterceptor(i grpc.UnaryServerInterceptor)` | append a unary interceptor (repeatable) |
| `WithServerOptions(options ...grpc.ServerOption)` | pass native gRPC server options |

## Configuration

`grpcinitr.Config` owns `Reflection` and `HealthCheck`; the application owns
the port and any aggregation of server instances. `WithConfig` applies feature
values, and later options override them. `WithInterceptor` and
`WithServerOptions` are deliberate native gRPC escape hatches.

Options are sealed: callers can compose the named options but cannot mutate
grpcinitr's private construction state.

## Lifecycle

`Close(ctx)` gracefully stops the server, falling back to a hard stop if the
App's shutdown context expires. It does not create a separate deadline or
handle process signals. Safe to call after `Serve` returns. Implementing
`apin.Closer`, it slots directly into `apin.App.Track`.

Track order matters: track the server shell *after* the databases it depends
on, so reverse-order shutdown stops serving before closing connections.
