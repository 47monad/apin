# zapinitr

[Zap](https://pkg.go.dev/go.uber.org/zap) logger initializer. Its
initializer-owned configuration and shell keep this module independent of
the apin root module.

```bash
go get github.com/47monad/apin/initrs/zapinitr
```

## Usage

Applications define aggregate configuration from only the initrs they select:

```go
type serviceConfig struct {
	Logging zapinitr.Config `json:"logging" yaml:"logging"`
}

loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
if err != nil {
	return err
}
app.RegisterLogger(loggerShell.Logger)
app.Track(loggerShell)
```

Individual options are also available; later options override configuration:

```go
loggerShell, err := zapinitr.New(ctx,
	zapinitr.WithConfig(&cfg.Logging),
	zapinitr.WithLevel("debug"),
)
```

## Config and options

`Config` currently contains `Level` (`"debug"`, `"info"`, `"warn"`,
`"error"`, or `"fatal"`). `WithConfig(*Config)` applies it, while
`WithLevel` can override it. An empty level uses zap's production default;
invalid levels fail construction.

The shell exposes a `logr.Logger` and implements
`Close(context.Context) error`, which flushes the native zap logger. The
application decides when the shell is closed by tracking it with `App.Track`.
