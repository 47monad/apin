# prominitr

Prometheus initr. Returns a shell with a fresh
[`prometheus.Registry`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Registry)
for the service's metrics — instead of the global default registry.

```bash
go get github.com/47monad/apin/initrs/prominitr
```

## Usage

```go
type serviceConfig struct {
	Name       string            `json:"name" yaml:"name"`
	Prometheus *prominitr.Config `json:"prometheus" yaml:"prometheus"`
}

promShell, err := prominitr.New(ctx, prominitr.WithConfig(cfg.Prometheus))
```

It is also valid to create it without options — the registry needs no
connection:

```go
promShell, err := prominitr.New(ctx)
```

## Shell

```go
type Shell struct {
	Registry               *prometheus.Registry
	GRPCServerInterceptor grpc.UnaryServerInterceptor
	GRPCServerMetrics     *grpcprom.ServerMetrics
}
```

Register collectors on `promShell.Registry` and expose them through any HTTP
handler (`promhttp.HandlerFor(shell.Registry, ...)`).

## Options

| Option | Description |
|---|---|
| `WithConfig(*prominitr.Config)` | apply an initializer-owned config section |
| `WithGRPCMetrics(bool)` | enable/disable gRPC metrics configuration |

`Config` contains the optional `GRPCMetrics` toggle. When enabled, the shell
registers gRPC metrics in its registry and exposes both the unary interceptor
and native `*grpcprom.ServerMetrics` handle. Applications can pass the
interceptor to the selected gRPC server initializer with
`grpcinitr.WithInterceptor`; `prominitr` itself does not depend on `grpcinitr`.
`WithPromMonitoring(reg)` remains available for applications that need to
attach instrumentation to another registry.

Options are sealed: callers can compose the named options but cannot mutate
prominitr's private construction state.

## Lifecycle

`Close(ctx)` is a no-op because the shell holds no external resources. Its
context-aware close method lets applications use their own lifecycle policy.
