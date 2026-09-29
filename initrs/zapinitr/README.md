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
app := apin.New(apin.WithLogger(loggerShell.Logger))
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

Options are sealed: callers can compose named options but cannot mutate
zapinitr's private construction state. `WithNativeConfig(func(*zap.Config) error)`
is the deliberate escape hatch for zap output, encoding, sampling, and other
native configuration.

The shell exposes two handles over the same logger core:

- `Shell.Logger` is the `logr.Logger` view, for components that accept a
  `logr.Logger` (for example `apin.WithLogger`).
- `Shell.Zap` is the native `*zap.Logger`, for zap-specific features such as
  structured fields or third-party integrations.

The shell implements `Close(context.Context) error`, which flushes the native
zap logger; sync errors from sinks that cannot be synced (stdout, stderr,
pipes, terminals) are treated as benign. The application decides when to close
it by tracking the shell with `App.Track`.
