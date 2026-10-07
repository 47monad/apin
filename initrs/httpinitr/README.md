# httpinitr

`httpinitr` owns basic HTTP server configuration and construction without
depending on the apin root module or another initializer.

```bash
go get github.com/47monad/apin/initrs/httpinitr
```

Applications can compose its config with only the initializers they select:

```go
type serviceConfig struct {
	HTTP httpinitr.Config `json:"http" yaml:"http"`
}

mux := http.NewServeMux()
shell, err := httpinitr.NewServer(ctx, cfg.HTTP.Servers["api"],
	httpinitr.WithHandler(mux),
)
if err != nil {
	return err
}
listener, err := shell.Listen()
if err != nil {
	return err
}
app.Track(shell)
return app.Run(ctx, func(ctx context.Context) error {
	return shell.Serve(ctx, listener)
})
```

`Config.Servers` contains named `ServerConfig` entries. Each server listens on
an empty host (all interfaces) and port `4747` by default; ports must be `1`
through `65535`. Later options win.

Timeouts are expressed in seconds in `ServerConfig` and as `time.Duration` in
options. `ReadHeaderTimeout` defaults to 10s, `IdleTimeout` to 120s, and
`ReadTimeout` / `WriteTimeout` default to zero so streaming stays unbounded. An
explicit zero disables a timeout, and `MaxHeaderBytes` of zero selects
`http.DefaultMaxHeaderBytes`. Negative values are rejected.

`ServerShell.Close(ctx)` gracefully shuts down, falling back to closing
connections if the context deadline expires. Track the shell with the
application so the application owns shutdown policy. `WithBaseContext` and
`WithNativeServer` are escape hatches for the native `http.Server`.

## Options

| Option | Description |
|---|---|
| `WithConfig(*httpinitr.ServerConfig)` | apply an initializer-owned config section |
| `WithHost(host string)` | listening host; empty means all interfaces |
| `WithPort(port int)` | listening port; zero selects the default `4747` |
| `WithHandler(http.Handler)` | handler; nil uses an empty mux |
| `WithReadHeaderTimeout(time.Duration)` | header read timeout; default `10s` |
| `WithReadTimeout(time.Duration)` | whole-request timeout; default `0` (none) |
| `WithWriteTimeout(time.Duration)` | response write timeout; default `0` (none) |
| `WithIdleTimeout(time.Duration)` | keep-alive idle timeout; default `120s` |
| `WithMaxHeaderBytes(bytes int)` | header size limit; zero selects `http.DefaultMaxHeaderBytes` |
| `WithBaseContext(func(net.Listener) context.Context)` | native base-context hook |
| `WithNativeServer(func(*http.Server) error)` | native server escape hatch |

Options are sealed: callers can compose the named options but cannot mutate
httpinitr's private construction state.

Create and track one shell per selected entry in `Config.Servers`; application
code does not need another HTTP configuration struct.
