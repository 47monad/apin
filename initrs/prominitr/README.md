# prominitr

Prometheus initr. Returns a shell with a fresh
[`prometheus.Registry`](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#Registry)
for the service's metrics — instead of the global default registry. gRPC
instrumentation lives in the separate
[`promgrpcinitr`](../promgrpcinitr) module, so this module carries no gRPC or
OpenTelemetry dependencies.

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
	Registry *prometheus.Registry
}
```

Register collectors on `promShell.Registry` and expose them through any HTTP
handler (`promhttp.HandlerFor(shell.Registry, ...)`).

## Options

| Option | Description |
|---|---|
| `WithConfig(*prominitr.Config)` | apply an initializer-owned config section |
| `WithGoCollector(bool)` | register the standard Go runtime collectors |
| `WithProcessCollector(bool)` | register the standard process collectors |

`Config` contains optional `GoCollector` and `ProcessCollector` toggles; both
default to off. Later options override config values.

Options are sealed: callers can compose the named options but cannot mutate
prominitr's private construction state.

## Lifecycle

`Close(ctx)` is a no-op because the shell holds no external resources. Its
context-aware close method lets applications use their own lifecycle policy.
Implementing `apin.Closer`, it slots directly into `apin.App.Track`.
