# apin

[![License: MIT](https://img.shields.io/badge/MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/47monad/apin)](https://goreportcard.com/report/github.com/47monad/apin)
[![Go Reference](https://pkg.go.dev/badge/github.com/47monad/apin.svg)](https://pkg.go.dev/github.com/47monad/apin)

> **apin** (/æpin/) - short for "**ap**p **in**itializer" - Uniform starting points for Go microservices.

## Overview

Apin provides a uniform way to bootstrap the infrastructure services a
microservice needs. Each service (`pginitr`, `mongoinitr`, `rmqinitr`, ...)
is a separate Go module that turns a [zaal](https://github.com/47monad/zaal)
config section into a ready-to-use **Shell** — with functional options for
programmatic overrides.

You install only the initrs your service actually needs:

```bash
go get github.com/47monad/apin
go get github.com/47monad/apin/initrs/pginitr
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	"github.com/47monad/apin"
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/47monad/zaal"
)

func main() {
	ctx := context.Background()

	cfg, err := zaal.New("config.json", ".env") // zaal parses the config file
	if err != nil {
		log.Fatal(err)
	}

	loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
	if err != nil {
		log.Fatal(err)
	}

	app := apin.NewApp(apin.WithLogger(loggerShell.Logger))

	dbShell, err := pginitr.New(ctx,
		pginitr.WithConfig(cfg.Postgres), // config file values...
		pginitr.WithSSLMode("require"),   // ...overridden field by field
	)
	if err != nil {
		log.Fatal(err)
	}
	app.Track(dbShell)

	grpcCfg := cfg.GRPC.Servers["api"]
	srvShell, err := grpcinitr.New(ctx,
		grpcinitr.WithConfig(&grpcCfg),
		grpcinitr.WithRunnable(func(s *grpc.Server) {
			pb.RegisterUserServiceServer(s, &userServer{db: dbShell})
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	app.Track(srvShell)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcCfg.Port))
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(ctx, func(ctx context.Context) error {
		return srvShell.Serve(ctx, lis) // serves until app shutdown
	}); err != nil {
		loggerShell.Logger.Error(err, "application failed")
	}
}
```

A returned shell is ready to use (connections are verified at `New`), and
`app.Run` blocks until a runnable fails, the context is cancelled, or
SIGINT/SIGTERM — then closes every tracked shell in reverse initialization
order.

## The Shell Law

Every initr follows the same contract, so any service reads the same way:

1. **Shells.** Each initr returns a `Shell` — the ready-to-use handle for its
   service (e.g. `pginitr.Shell`, `rmqinitr.Shell`). Cross-cutting shells
   (logging) return `apin.LoggerShell`, so consumers are logger-agnostic.
2. **Construction.** `New(ctx, opts ...Option)` and `MustNew(ctx, opts...)`.
   `New` connects eagerly and fails fast on missing configuration or dial
   errors.
3. **Options.** Functional options (`Option func(*Store) error`) are applied
   in order; later options win.
4. **Config entry point.** `WithConfig(*zaal.XConfig)` is the config-file
   path. Individual `With*` options override single fields on top of it:
   ```go
   pginitr.New(ctx, pginitr.WithConfig(cfg.Postgres), pginitr.WithPort(6543))
   ```
5. **Cleanup.** Every `Shell` implements `apin.Closer` (`Close(ctx) error`).
6. **Library discipline.** Defaults are applied inside the initr (never by
   mutating the caller's config), panics only ever come from `MustNew`, and
   initrs never write to stderr — pass a logger with `WithLogger`.

### Naming conventions

| Form | Meaning |
|---|---|
| `WithConfig(cfg)` | apply a zaal config section |
| `With*` | set a scalar / toggle / composite |
| `Add*` | append to a list (e.g. `AddInterceptor`) |

## Initrs

| Module | Shell | Notes |
|---|---|---|
| `initrs/pginitr` | `Shell{Mode, Pool, Conn}` | pool (default) or single connection; `DB()` gives a mode-independent query surface |
| `initrs/mongoinitr` | `Shell{Client, DB}` | ping-checked connection |
| `initrs/etcdinitr` | `Shell{Client}` | |
| `initrs/rmqinitr` | `Shell` | auto-reconnecting connection/channel; `WaitForHealth` |
| `initrs/grpcinitr` | `ServerShell{Server, HealthServer}` | health check + reflection toggles |
| `initrs/prominitr` | `Shell{Registry}` | |
| `initrs/zapinitr` | `apin.LoggerShell` | any logger initr returns the same shell |

## Graceful Shutdown

`apin.App` owns the shutdown:

```go
app := apin.NewApp(apin.WithLogger(loggerShell.Logger))
defer app.Close(context.Background()) // manual lifecycle control
```

- `app.Track(shell...)` — record shells for cleanup (call it right after each `New`)
- `app.Run(ctx, runnables...)` — start serving; on SIGINT/SIGTERM or context
  cancellation, runnables are cancelled and shells closed in reverse order,
  bounded by a shutdown timeout (default 30s)
- A second signal forces an immediate exit

`app.Close(ctx)` alone closes tracked shells in reverse order — useful for
tests or custom lifecycles.

The `runner` package still works on its own for concurrent multi-server
setups (`runner.AddGRPCServer`, `AddHTTPServer`, `AddHealthCheck`) — pass
`runner.Run` as the App's runnable (wrap it in a `func(ctx)` that ignores the
context), or use `ServerShell.Serve` for the single-server case shown above.

## Configuration

Configs are plain structs from [zaal](https://github.com/47monad/zaal), so
`WithConfig` is explicit at every call site — you always know where a value
came from. Programmatic `With*` options compose with it, field by field:

```go
pginitr.New(ctx,
	pginitr.WithConfig(cfg.Postgres), // from the config file
	pginitr.WithMode(pginitr.ModeConn), // override: single connection
	pginitr.WithDBName("settings"),
)
```

Initrs can also be configured without any config file, using options only.

## Repository Layout

- `common.go`, `app.go` — apin core (`LoggerShell`, `Closer`, `App`)
- `closr/` — the `Closer` alias, kept for compatibility
- `runner/` — errgroup-based concurrent runner
- `initrs/` — one module per service initr

## Contributing

When adding an initr, follow the [Shell Law](#the-shell-law): a `Shell` type,
`New`/`MustNew` with variadic options, `WithConfig` mapping the zaal section,
defaults and fail-fast validation inside `New`, and a `Close(ctx) error`.
Add the module to `go.work`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
