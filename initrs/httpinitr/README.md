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
shell, err := httpinitr.New(ctx,
	httpinitr.WithConfig(&cfg.HTTP),
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

`Config.Port` defaults to `4747` and accepts ports from `1` through `65535`.
`WithPort` overrides it; later options win. `WithHandler` sets the handler,
and `Server` exposes the native `*http.Server` for additional configuration.
`ServerShell.Close(ctx)` gracefully shuts down, falling back to closing
connections if the context deadline expires. Track the shell with the
application so the application owns shutdown policy.

Options are sealed: callers can compose the named options but cannot mutate
httpinitr's private construction state. Advanced native HTTP server settings
remain available through `ServerShell.Server`.

For multiple named listeners, keep a `map[string]httpinitr.Config` in the
application aggregate and create one tracked shell per entry. The runnable
example preserves the former named-server configuration shape this way.
