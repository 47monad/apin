# mongoinitr

MongoDB initr. Returns a ready-to-use shell holding a
[`mongo.Client`](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/mongo#Client)
and, when a database name is set, its default `*mongo.Database`.

```bash
go get github.com/47monad/apin/initrs/mongoinitr
```

## Usage

```go
// Applications compose only the configuration they use.
type serviceConfig struct {
	Name  string             `json:"name" yaml:"name"`
	Mongo *mongoinitr.Config `json:"mongo" yaml:"mongo"`
}

var cfg serviceConfig
if err := config.Load("service.yaml", ".env", &cfg); err != nil {
	return err
}

dbShell, err := mongoinitr.New(ctx, mongoinitr.WithConfig(cfg.Mongo))

// config file + overrides
dbShell, err := mongoinitr.New(ctx,
	mongoinitr.WithConfig(cfg.Mongo),
	mongoinitr.WithPingTimeout(3*time.Second),
)

// no config file, fully programmatic
dbShell, err := mongoinitr.New(ctx,
	mongoinitr.WithURI("mongodb://localhost:27017"),
	mongoinitr.WithDBName("settings"),
)
```

`New` verifies the connection with a ping before returning — a returned shell
is ready to use.

## Shell

```go
type Shell struct {
	Client *mongo.Client
	DB     *mongo.Database // nil unless a DB name was set
}
```

## Options

| Option | Description |
|---|---|
| `WithConfig(*mongoinitr.Config)` | apply initializer-owned configuration |
| `WithURI(uri string)` | apply a mongodb connection URI (applies all URI options to the driver) |
| `WithTimeout(d time.Duration)` | driver connect timeout |
| `WithDBName(name string)` | default database of the returned shell |
| `WithPingTimeout(d time.Duration)` | readiness ping timeout; defaults to `10s` |

## Config mapping

`Config` is decoded by the application's loader, not by mongoinitr. `WithConfig`
applies the URI and database name; later options override either value. The
initializer validates the resolved driver options before attempting a
connection. `PingTimeout` defaults to `10s`.

## Lifecycle

`Close(ctx)` disconnects the client. Implementing `apin.Closer`, it slots
directly into `apin.App.Track`.
