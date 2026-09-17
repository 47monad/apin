# mongoinitr

MongoDB initr. Returns a ready-to-use shell holding a
[`mongo.Client`](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/mongo#Client)
and, when a database name is set, its default `*mongo.Database`.

```bash
go get github.com/47monad/apin/initrs/mongoinitr
```

## Usage

```go
// config-file driven
dbShell, err := mongoinitr.New(ctx, mongoinitr.WithConfig(cfg.Mongodb))

// config file + overrides
dbShell, err := mongoinitr.New(ctx,
	mongoinitr.WithConfig(cfg.Mongodb),
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
| `WithConfig(*zaal.MongodbConfig)` | apply a zaal config section (entry point for config-file setups) |
| `WithURI(uri string)` | apply a mongodb connection URI (applies all URI options to the driver) |
| `WithTimeout(d time.Duration)` | driver connect timeout |
| `WithDBName(name string)` | default database of the returned shell |
| `WithPingTimeout(d time.Duration)` | readiness ping timeout; defaults to `10s` |

## Config mapping

`WithConfig` maps `*zaal.MongodbConfig`: `URI` and `DBName`. Any of these can
be overridden by a later option.

## Lifecycle

`Close(ctx)` disconnects the client. Implementing `apin.Closer`, it slots
directly into `apin.App.Track`.
