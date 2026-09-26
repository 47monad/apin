# etcdinitr

etcd initr. Returns a ready-to-use shell holding an etcd
[`clientv3.Client`](https://pkg.go.dev/go.etcd.io/etcd/client/v3#Client).

```bash
go get github.com/47monad/apin/initrs/etcdinitr
```

## Usage

```go
// Applications compose only the configuration they use.
type serviceConfig struct {
	Name string            `json:"name" yaml:"name"`
	Etcd *etcdinitr.Config `json:"etcd" yaml:"etcd"`
}

var cfg serviceConfig
if err := config.Load("service.yaml", ".env", &cfg); err != nil {
	return err
}

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
| `WithConfig(*etcdinitr.Config)` | apply initializer-owned configuration |
| `WithEndpoints(endpoints []string)` | etcd endpoints |
| `WithUsername(username string)` | auth username |
| `WithPassword(password string)` | auth password |
| `WithTimeout(d time.Duration)` | dial timeout |
| `WithNativeConfig(func(*clientv3.Config) error)` | configure native etcd client settings not represented by the named options |

## Config mapping

`Config` is decoded by the application's loader, not by etcdinitr. `Endpoints`
remains comma-separated and `Timeout` is an optional positive number of
seconds. An omitted timeout leaves the native client's timeout unset, while an
explicit zero or negative timeout is rejected. Later options override
individual configured values.

Options are sealed: callers can compose the named options but cannot mutate
etcdinitr's private construction state. Native client configuration is the
deliberate driver escape hatch.

## Lifecycle

`Close(ctx)` closes the client. Implementing `apin.Closer`, it slots directly
into `apin.App.Track`.
