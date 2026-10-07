# redisinitr

Redis initr. Returns a shell holding a go-redis
[`redis.UniversalClient`](https://pkg.go.dev/github.com/redis/go-redis/v9#UniversalClient)
covering standalone, Sentinel, and cluster deployments.

```bash
go get github.com/47monad/apin/initrs/redisinitr
```

## Usage

```go
type serviceConfig struct {
	Name  string             `json:"name" yaml:"name"`
	Redis *redisinitr.Config `json:"redis" yaml:"redis"`
}

redisShell, err := redisinitr.New(ctx, redisinitr.WithConfig(cfg.Redis))
if err != nil {
	return err
}
readyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
if err := redisShell.Ready(readyCtx); err != nil {
	return err
}

// no config file, fully programmatic
redisShell, err := redisinitr.New(ctx,
	redisinitr.WithAddresses([]string{"localhost:6379"}),
	redisinitr.WithPassword("secret"),
)
```

`New` validates the configuration and constructs the client lazily without
connecting. `Ready(ctx)` pings the server; pass a deadline-bearing context to
bound the check.

## Shell

```go
type Shell struct {
	Client redis.UniversalClient
}
```

One address selects a standalone client, several select a cluster client, and
`MasterName` selects Sentinel. `Ready(ctx)` pings the server and `Close(ctx)`
releases the client.

## Options

| Option | Description |
|---|---|
| `WithConfig(*redisinitr.Config)` | apply an initializer-owned config section |
| `WithAddresses(addresses []string)` | one or more Redis endpoints |
| `WithUsername(username string)` | ACL username |
| `WithPassword(password string)` | password |
| `WithDatabase(database int)` | logical database (standalone/Sentinel) |
| `WithMasterName(master string)` | Sentinel master name |
| `WithNativeOptions(func(*redis.UniversalOptions) error)` | native go-redis escape hatch |

## Config mapping

`Config` is decoded by the application's loader, not by redisinitr. `Addresses`
is a non-empty list (comma-separated when supplied through an environment
variable), and `Database` defaults to `0` and must not be negative. Later
options override configured values.

Options are sealed: callers can compose the named options but cannot mutate
redisinitr's private construction state. `WithNativeOptions` is the deliberate
driver escape hatch.

## Lifecycle

`Close(ctx)` closes the underlying client. Implementing `apin.Closer`, it slots
directly into `apin.App.Track`.
