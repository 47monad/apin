# apin

[![License: MIT](https://img.shields.io/badge/MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/47monad/apin)](https://goreportcard.com/report/github.com/47monad/apin)
[![Go Reference](https://pkg.go.dev/badge/github.com/47monad/apin.svg)](https://pkg.go.dev/github.com/47monad/apin)

> **apin** (/æpin/) - short for "**ap**p **in**itializer" - Uniform starting points for Go microservices.

## Overview

Apin provides shells for infrastructure services. Each initializer is a
separate Go module with functional options; applications select the
initializers they need and own any aggregate configuration type.

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
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/zapinitr"
)

type serviceConfig struct {
	Name     string                 `json:"name" yaml:"name"`
	Logging  zapinitr.Config        `json:"logging" yaml:"logging"`
	Postgres *pginitr.Config        `json:"postgres" yaml:"postgres"`
	GRPC     *grpcConfig            `json:"grpc" yaml:"grpc"`
}

type grpcConfig struct {
	Servers map[string]grpcServerConfig `json:"servers" yaml:"servers"`
}

type grpcServerConfig struct {
	Port     int               `json:"port" yaml:"port"`
	Features grpcinitr.Config `json:"features" yaml:"features"`
}

func main() {
	ctx := context.Background()

	var cfg serviceConfig
	if err := config.Load("config.json", "", &cfg); err != nil {
		log.Fatal(err)
	}
	// The logger shell provides the App's lifecycle logger.
	loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
	if err != nil {
		log.Fatal(err)
	}
	app := apin.New(apin.WithLogger(loggerShell.Logger))
	app.Track(loggerShell)

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
		grpcinitr.WithConfig(&grpcCfg.Features),
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

A complete runnable version of this lives in
[`examples/grpcsvc`](examples/grpcsvc), including its application-owned
aggregate configuration.

## The Shell Law

Every initr follows the same contract, so any service reads the same way:

1. **Shells.** Each initr returns a `Shell` — the ready-to-use handle for its
   service (e.g. `pginitr.Shell`, `rmqinitr.Shell`). Cross-cutting shells
   (logging) return an initializer-owned shell exposing `logr.Logger`.
2. **Construction.** `New(ctx, opts ...Option)` and `MustNew(ctx, opts...)`.
   `New` connects eagerly and fails fast on missing configuration or dial
   errors.
3. **Options.** Functional options (`Option func(*Store) error`) are applied
   in order; later options win.
4. **Config entry point.** Each initializer can accept its own configuration
	type. Applications choose the fields they need and individual `With*`
	options override single fields:
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
| `WithConfig(cfg)` | apply initializer configuration |
| `With*` | set a scalar / toggle / composite |
| `Add*` | append to a list |

## Initrs

| Module | Shell | Notes |
|---|---|---|
| [`initrs/pginitr`](initrs/pginitr) | `Shell{Mode, Pool, Conn}` | pool (default) or single connection; `DB()` gives a mode-independent query surface |
| [`initrs/mongoinitr`](initrs/mongoinitr) | `Shell{Client, DB}` | ping-checked connection |
| [`initrs/etcdinitr`](initrs/etcdinitr) | `Shell{Client}` | |
| [`initrs/rmqinitr`](initrs/rmqinitr) | `Shell` | auto-reconnecting connection/channel; `WaitForHealth` |
| [`initrs/grpcinitr`](initrs/grpcinitr) | `ServerShell{Server, HealthServer}` | health/reflection, `RunHealthCheck`, ctx-aware `Serve` |
| [`initrs/prominitr`](initrs/prominitr) | `Shell{Registry, GRPCServerInterceptor, GRPCServerMetrics}` | optional gRPC instrumentation adapter |
| [`initrs/zapinitr`](initrs/zapinitr) | `zapinitr.Shell` | initializer-owned logger shell |

## Graceful Shutdown

`apin.App` owns the shutdown:

```go
app := apin.New(apin.WithLogger(loggerShell.Logger))
app.Track(loggerShell)
defer app.Close(context.Background()) // manual lifecycle control
```

- `app.Track(shell...)` — record shells for cleanup (call it right after each `New`)
- `app.Run(ctx, runnables...)` — start serving; on SIGINT/SIGTERM or context
  cancellation, runnables are cancelled and shells closed in reverse order,
  bounded by a shutdown timeout (default 30s, `app.SetShutdownTimeout` to tune)
- A second signal cancels cleanup more aggressively; process termination
  remains the application's decision.

`app.Close(ctx)` alone closes tracked shells in reverse order — useful for
tests or custom lifecycles.

Use each initializer shell to serve and stop its native resource. The App
cancels runnables on shutdown, then closes tracked shells in reverse order
with the same shutdown context and deadline. For example, `httpinitr` owns
HTTP server shutdown and `grpcinitr` owns gRPC graceful-stop behavior; neither
initializer installs process signal handlers.

## Configuration

The standalone `config` module loads JSON or YAML into an application-owned
aggregate. PostgreSQL configuration belongs to `pginitr.Config`; the
application includes only the initializer types it selects:

```go
type serviceConfig struct {
	Name     string          `json:"name" yaml:"name"`
	Postgres *pginitr.Config `json:"postgres" yaml:"postgres"`
}

var cfg serviceConfig
if err := config.Load("config.json", ".env", &cfg); err != nil {
	log.Fatal(err)
}

pginitr.New(ctx,
	pginitr.WithConfig(cfg.Postgres),
	pginitr.WithMode(pginitr.ModeConn), // override: single connection
	pginitr.WithDBName("settings"),
)
```

The loader reads `.env` values without changing the process environment.
Precedence is process environment > `.env` file > configuration file.

An initializer can also be configured entirely through options.

## Repository Layout

- `common.go`, `app.go` — minimal lifecycle module (`App`, `Closer`, `Runnable`)
- `config/` — standalone JSON/YAML loading and environment overlays
- `closr/` — the `Closer` alias, kept for compatibility
- `initrs/` — one module per service initr
- `examples/` — runnable example services (see `examples/grpcsvc`)

## Contributing

When adding an initr, follow the [Shell Law](#the-shell-law): a `Shell` type,
`New`/`MustNew` with variadic options, initializer-owned configuration,
defaults and fail-fast validation inside `New`, and a
`Close(ctx) error`. Add the module to `go.work`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
