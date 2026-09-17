# zapinitr

[Zap](https://pkg.go.dev/go.uber.org/zap) logger initr. Returns an
`apin.LoggerShell` — the cross-cutting shell every logger initr returns, so
consumers are logger-agnostic.

```bash
go get github.com/47monad/apin/initrs/zapinitr
```

## Usage

```go
loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))

// config file + overrides
loggerShell, err := zapinitr.New(ctx,
	zapinitr.WithConfig(&cfg.Logging),
	zapinitr.WithLevel("debug"),
)

// no config file
loggerShell, err := zapinitr.New(ctx)
```

## Shell

```go
// apin (core) — uniform for any logger initr
type LoggerShell struct {
	Logger logr.Logger
}
```

The shell wraps zap in a `logr.Logger` (`zapr`), so the rest of the app and
initrs depend on `logr`, never on zap directly. Callers that need raw zap can
adapt `zaplogr.Logger` themselves.

## Options

| Option | Description |
|---|---|
| `WithConfig(*zaal.LoggingConfig)` | apply a zaal config section (entry point for config-file setups) |
| `WithLevel(level string)` | log level (`"debug"`, `"info"`, ...); invalid values fail `New`; defaults to zap's production default |

## Config mapping

`WithConfig` maps `*zaal.LoggingConfig.Level`. It can be overridden by a
later option.

## Production preset

The logger is built from zap's production config with `AddCallerSkip(1)` (so
callers through the `logr` wrapper report correctly), a fixed time layout,
and stacktraces hidden.

## Lifecycle

`LoggerShell` holds no external resources and does not implement
`apin.Closer` — there is nothing to close, so it is not tracked by
`apin.App`.
