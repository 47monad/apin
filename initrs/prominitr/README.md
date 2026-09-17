# prominitr

Prometheus initr. Returns a shell with a fresh
[`prometheus.Registry`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Registry)
for the service's metrics — instead of the global default registry.

```bash
go get github.com/47monad/apin/initrs/prominitr
```

## Usage

```go
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
	Registry *prometheus.Registry
}
```

Register collectors on `promShell.Registry` and expose them through any HTTP
handler (`promhttp.HandlerFor(shell.Registry, ...)`).

## Options

| Option | Description |
|---|---|
| `WithConfig(*zaal.PrometheusConfig)` | apply a zaal config section (entry point for config-file setups) |

`GRPCMetrics` from the config is read into the store; wiring it into
`grpcinitr` is still pending (see the TODO in `opts.go`).

## Lifecycle

`Close(ctx)` is a no-op (the shell holds no external resources) but is
provided so it implements `apin.Closer` and slots into `apin.App.Track` for
uniformity.
