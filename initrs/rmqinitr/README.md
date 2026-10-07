# rmqinitr

RabbitMQ initr. Returns a shell with an auto-reconnecting AMQP connection,
guarded by a background reconnect loop with exponential backoff. Channels are
caller-owned and never drive reconnection.

```bash
go get github.com/47monad/apin/initrs/rmqinitr
```

## Usage

```go
// application-owned aggregate config
type serviceConfig struct {
	Name     string           `json:"name" yaml:"name"`
	RabbitMQ *rmqinitr.Config `json:"rabbitmq" yaml:"rabbitmq"`
}

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
shell.IsHealthy()             // connection established and shell not closed
shell.WaitForHealth(ctx)      // blocks until healthy or ctx done
ch, err := shell.NewChannel() // caller-owned; ErrNotHealthy / ErrShellClosed otherwise
conn, err := shell.GetConn()
```

`NewChannel` opens a channel on the current connection and returns it to the
caller, who must close it. Closing a channel never marks the shell unhealthy or
triggers a reconnect; only the connection's lifecycle drives the reconnect
loop. After a reconnect, call `NewChannel` again to get a channel on the new
connection. Use `GetConn` for native driver access.

## Options

| Option | Description |
|---|---|
| `WithConfig(*rmqinitr.Config)` | apply an initializer-owned config section |
| `WithURI(uri string)` | amqp connection URI |
| `WithMinRetryInterval(d time.Duration)` | initial reconnect backoff; defaults to `1s` |
| `WithMaxRetryInterval(d time.Duration)` | backoff cap; defaults to `30s`; must not be lower than min |
| `WithLazyConnect()` | defer the first connection to the background loop; `New` then succeeds without reaching the broker |
| `WithLogger(logr.Logger)` | logger for connection lifecycle events; defaults to discarding |
| `WithNativeDialConfig(func(*amqp.Config) error)` | configure native AMQP transport and handshake settings |

## Config mapping

`WithConfig` maps `*rmqinitr.Config`: `URI`, and `MinRetryInterval` /
`MaxRetryInterval` (seconds). Omitted retry intervals default to 1 and 30
seconds. It never mutates the config you pass in. Any
value can be overridden by a later option.

Options are sealed: callers can compose named options but cannot mutate
rmqinitr's private construction state. `WithNativeDialConfig` is the deliberate
driver escape hatch; without it, `New` retains the driver's standard `Dial`
behavior.

## Lifecycle

`Close(ctx)` is idempotent: it stops the reconnect loop, cancels and interrupts
an in-progress network dial/AMQP handshake, and closes the connection.
It waits only until `ctx` is done; if native resource cleanup is still running,
that cleanup continues in the background. A custom `WithNativeDialConfig` dial
callback has no context parameter, but shutdown still returns by the supplied
deadline; a callback that never returns may leave its own goroutine running.
Implementing `apin.Closer`, it slots directly into `apin.App.Track`.
