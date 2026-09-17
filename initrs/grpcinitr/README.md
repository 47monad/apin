# grpcinitr

gRPC server initr. Returns a shell holding a configured
[`grpc.Server`](https://pkg.go.dev/google.golang.org/grpc#Server), with
optional health checking and reflection, ready for registration and serving.

```bash
go get github.com/47monad/apin/initrs/grpcinitr
```

## Usage

```go
// config-file driven
srvShell, err := grpcinitr.New(ctx, grpcinitr.WithConfig(&grpcCfg))

// config file + service registration + interceptors
srvShell, err := grpcinitr.New(ctx,
	grpcinitr.WithConfig(&grpcCfg),
	grpcinitr.WithRunnable(func(s *grpc.Server) {
		pb.RegisterUserServiceServer(s, &userServer{db: dbShell})
	}),
	grpcinitr.WithInterceptor(authInterceptor),
)
```

## Serving

`Serve` is context-aware and directly usable as an `apin.App` runnable —
serving stops gracefully when the app shuts down:

```go
lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcCfg.Port))

app.Run(ctx, func(ctx context.Context) error {
	return srvShell.Serve(ctx, lis) // graceful stop on app shutdown
})
```

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
| `WithConfig(*manifest.GRPCServerConfig)` | apply a manifest config section (entry point for config-file setups) |
| `WithReflection(enabled bool)` | register the gRPC reflection service |
| `WithHealthCheck(enabled bool)` | register the standard gRPC health checking service |
| `WithRunnable(fn func(*grpc.Server))` | bootstrap logic run against the created server — where services get registered |
| `WithInterceptor(i grpc.UnaryServerInterceptor)` | append a unary interceptor (repeatable) |

## Config mapping

`WithConfig` maps `*manifest.GRPCServerConfig.Features`: `Reflection` and
`HealthCheck`. (The `Port` is used where you decide to listen — see serving
above — the initr itself does not bind.) Any value can be overridden by a
later option.

## Lifecycle

`Close(ctx)` gracefully stops the server, falling back to a hard stop if the
context expires — bounded, so it cannot hang `apin.App` shutdown. Safe to
call after `Serve` already shut the server down. Implementing `apin.Closer`,
it slots directly into `apin.App.Track`.

Track order matters: track the server shell *after* the databases it depends
on, so reverse-order shutdown stops serving before closing connections.
