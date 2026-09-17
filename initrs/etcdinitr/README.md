# etcdinitr

etcd initr. Returns a ready-to-use shell holding an etcd
[`clientv3.Client`](https://pkg.go.dev/go.etcd.io/etcd/client/v3#Client).

```bash
go get github.com/47monad/apin/initrs/etcdinitr
```

## Usage

```go
// config-file driven
etcdShell, err := etcdinitr.New(ctx, etcdinitr.WithConfig(cfg.Etcd))

// config file + overrides
etcdShell, err := etcdinitr.New(ctx,
	etcdinitr.WithConfig(cfg.Etcd),
	etcdinitr.WithTimeout(5*time.Second),
)

// no config file, fully programmatic
etcdShell, err := etcdinitr.New(ctx,
	etcdinitr.WithEndpoints([]string{"localhost:2379"}),
	etcdinitr.WithUsername("root"),
	etcdinitr.WithPassword("secret"),
)
```

`New` connects eagerly and fails fast on dial errors — a returned shell is
ready to use.

## Shell

```go
type Shell struct {
	Client *clientv3.Client
}
```

## Options

| Option | Description |
|---|---|
| `WithConfig(*zaal.EtcdConfig)` | apply a zaal config section (entry point for config-file setups) |
| `WithEndpoints(endpoints []string)` | etcd endpoints |
| `WithUsername(username string)` | auth username |
| `WithPassword(password string)` | auth password |
| `WithTimeout(d time.Duration)` | dial timeout |

## Config mapping

`WithConfig` maps `*zaal.EtcdConfig`: comma-separated `Endpoints`,
`Username`, `Password`, and `Timeout` (seconds). Any of these can be
overridden by a later option.

## Lifecycle

`Close(ctx)` closes the client. Implementing `apin.Closer`, it slots directly
into `apin.App.Track`.
