# rmqinitr

RabbitMQ initr. Returns a shell with an auto-reconnecting AMQP
connection/channel, guarded by a background reconnect loop with exponential
backoff.

```bash
go get github.com/47monad/apin/initrs/rmqinitr
```

## Usage

```go
// config-file driven
mqShell, err := rmqinitr.New(ctx, rmqinitr.WithConfig(cfg.RabbitMQ))

// config file + overrides
mqShell, err := rmqinitr.New(ctx,
	rmqinitr.WithConfig(cfg.RabbitMQ),
	rmqinitr.WithLogger(logger),
)

// no config file, fully programmatic
mqShell, err := rmqinitr.New(ctx,
	rmqinitr.WithURI("amqp://guest:guest@localhost:5672/"),
)
```

By default `New` **connects synchronously** — a returned shell is connected
and healthy, and dial errors fail `New`. The background reconnect loop takes
over whenever the connection drops.

## Shell

```go
shell.IsHealthy()          // connection established and shell not closed
shell.WaitForHealth(ctx)   // blocks until healthy or ctx done
ch, err := shell.GetChannel() // ErrNotHealthy / ErrShellClosed otherwise
conn, err := shell.GetConn()
```

A single shared channel is exposed for convenience. Producers and consumers
with complex semantics should typically open their own channel via
`GetConn` — the reconnection shell keeps the connection alive, not your
channel state.

## Options

| Option | Description |
|---|---|
| `WithConfig(*manifest.RabbitMQConfig)` | apply a manifest config section (entry point for config-file setups) |
| `WithURI(uri string)` | amqp connection URI |
| `WithMinRetryInterval(d time.Duration)` | initial reconnect backoff; defaults to `1s` |
| `WithMaxRetryInterval(d time.Duration)` | backoff cap; defaults to `30s`; must not be lower than min |
| `WithLazyConnect()` | defer the first connection to the background loop; `New` then succeeds without reaching the broker |
| `WithLogger(logr.Logger)` | logger for connection lifecycle events; defaults to discarding |

## Config mapping

`WithConfig` maps `*manifest.RabbitMQConfig`: `URI`, and `MinRetryInterval` /
`MaxRetryInterval` (seconds). It never mutates the config you pass in. Any
value can be overridden by a later option.

## Lifecycle

`Close(ctx)` is idempotent: it stops the reconnect loop, waits for it, and
closes the connection/channel. Implementing `apin.Closer`, it slots directly
into `apin.App.Track`.
