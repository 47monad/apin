# pginitr

PostgreSQL initr. Returns a shell holding either a
[`pgxpool.Pool`](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool) (default)
or a single `*pgx.Conn`, selected by mode.

```bash
go get github.com/47monad/apin/initrs/pginitr
```

## Usage

```go
import "github.com/47monad/apin/config"

// The application owns the aggregate; pginitr owns its PostgreSQL section.
type serviceConfig struct {
	Name     string          `json:"name" yaml:"name"`
	Postgres *pginitr.Config `json:"postgres" yaml:"postgres"`
}

var cfg serviceConfig
if err := config.Load("config.json", ".env", &cfg); err != nil {
	panic(err)
}
dbShell, err := pginitr.New(ctx, pginitr.WithConfig(cfg.Postgres))

// config file + per-field overrides (later options win)
dbShell, err := pginitr.New(ctx,
	pginitr.WithConfig(cfg.Postgres),
	pginitr.WithSSLMode("require"),
	pginitr.WithPoolConfig(pginitr.PoolConfig{MaxConns: 20}),
)

// no config file, fully programmatic
dbShell, err := pginitr.New(ctx,
	pginitr.WithHost("localhost"),
	pginitr.WithPort(5432),
	pginitr.WithUser(url.UserPassword("postgres", "secret")),
	pginitr.WithDBName("settings"),
)
```

In pool mode, `New` validates the configuration and constructs the pool
without verifying database reachability; pool connections are lazy. In
single-connection mode, `New` opens a connection eagerly. Call
`dbShell.Ping(ctx)` to verify connectivity in either mode; give it a
deadline-bearing context to bound the probe. Calling `New` with no
configuration at all returns an error pointing at `WithConfig`/`WithURI`/
connection options.

## Shell

```go
type Shell struct {
	Mode Mode            // reports which strategy was selected
	Pool *pgxpool.Pool   // set when Mode == ModePool
	Conn *pgx.Conn       // set when Mode == ModeConn
}
```

`Ping(ctx)` verifies connectivity through the active pool or single
connection. It returns an error for an uninitialized shell.

Exactly one of `Pool` / `Conn` is non-nil. For mode-independent queries:

```go
q, err := dbShell.DB() // Querier: Exec, Query, QueryRow, Begin, SendBatch
tag, err := q.Exec(ctx, "insert into users (name) values ($1)", "alice")
```

Mode-specific capabilities stay on the fields: `Listen`/`WaitForNotification`
on `Conn`, `Acquire`/`Stat` on `Pool`. Note that single-connection mode has no
automatic reconnection — the pool is the default for a reason.

## Options

| Option | Description |
|---|---|
| `WithConfig(*pginitr.Config)` | apply initializer-owned PostgreSQL configuration |
| `WithURI(uri string)` | merge connection details from a postgres URI; query params preserved unless overridden later |
| `WithUser(*url.Userinfo)` | explicit credentials; takes precedence over URI-derived ones |
| `WithHost(host string)` | database host |
| `WithPort(port int)` | database port; composes with `WithHost`/URIs in any option order |
| `WithDBName(dbname string)` | database name |
| `WithSSLMode(sslmode string)` | sets the `sslmode` URI parameter |
| `WithParam(key, value string)` | sets a single URI parameter |
| `WithMode(mode Mode)` | `pginitr.ModePool` or `pginitr.ModeConn` |
| `WithPool()` | sugar for `WithMode(ModePool)` |
| `WithSingleConn()` | sugar for `WithMode(ModeConn)` |
| `WithPoolConfig(PoolConfig)` | pool tuning; only applied in pool mode |
| `WithNativePoolConfig(func(*pgxpool.Config) error)` | configure native pool features not represented by `PoolConfig` |
| `WithNativeConnConfig(func(*pgx.ConnConfig) error)` | configure native connection features not represented by `Config` |

`PoolConfig` fields (seconds for the time values, zero leaves pgxpool defaults):

```go
type PoolConfig struct {
	MaxConns            int
	MinConns            int
	MaxConnLifetime     int
	MaxConnIdleTime     int
	HealthCheckInterval int
}
```

## Config mapping

`WithConfig` maps `*pginitr.Config` field by field: `URI`, `Host`,
`Port` (int), `Username`/`Password`, `DBName`, `SSLMode` → `sslmode`,
`AppName` → `application_name`, `ConnTimeout` → `connect_timeout`, `Mode`
(`"pool"`/`"conn"`), and the `Pool` block. Any of these can be overridden by
a later option.

Options are sealed: callers can combine the named options but cannot mutate
pginitr's private construction state. Native pgx callbacks are the deliberate
escape hatch for driver settings not modeled by pginitr.

## Lifecycle

`Close(ctx)` closes the single connection (with error) or starts closing the
pool. pgx's pool close has no context-aware API, so if the supplied context
expires while checked-out connections are still active, `Close` returns the
context error and the pool continues draining in the background.
Implementing `apin.Closer`, it slots directly into `apin.App.Track`.
