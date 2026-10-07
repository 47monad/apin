# promgrpcinitr

gRPC instrumentation adapter for Prometheus. It is independent of
[`prominitr`](../prominitr) so that a plain Prometheus registry carries no gRPC
or OpenTelemetry dependencies; only services that instrument gRPC pull this
module.

```bash
go get github.com/47monad/apin/initrs/promgrpcinitr
```

```go
registry := prometheus.NewRegistry()
promShell, err := promgrpcinitr.New(ctx, promgrpcinitr.WithRegisterer(registry))
if err != nil {
	return err
}

grpcShell, err := grpcinitr.NewServer(ctx, cfg.GRPC.Servers["api"],
	grpcinitr.WithInterceptor(promShell.UnaryInterceptor),
	grpcinitr.WithStreamInterceptor(promShell.StreamInterceptor),
)
```

## Shell

```go
type Shell struct {
	UnaryInterceptor  grpc.UnaryServerInterceptor
	StreamInterceptor grpc.StreamServerInterceptor
	ServerMetrics     *grpcprom.ServerMetrics
}
```

`ServerMetrics` is the native
[`*grpcprom.ServerMetrics`](https://pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus#ServerMetrics)
handle for advanced use.

## Options

| Option | Description |
|---|---|
| `WithRegisterer(prometheus.Registerer)` | registerer that receives the metrics; defaults to `prometheus.DefaultRegisterer` |

Servers should use `prominitr`'s registry (or any `prometheus.Registerer`) as
the registerer, so gRPC metrics sit alongside the service's other metrics.

## Lifecycle

`Close(ctx)` is a no-op because the shell holds no external resources.
Implementing `apin.Closer`, it slots directly into `apin.App.Track`.
