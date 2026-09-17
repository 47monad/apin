# pginitr

PostgreSQL initr. Returns a ready-to-use shell holding either a
[`pgxpool.Pool`](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool) (default)
or a single `*pgx.Conn`, selected by mode.

```bash
go get github.com/47monad/apin/initrs/pginitr
```

## Usage

```go
// config-file driven
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

`New` connects eagerly and fails fast — a returned shell is ready to use.
Calling `New` with no configuration at all returns an error pointing at
`WithConfig`/`WithURI`/connection options.

## Shell

```go
type Shell struct {
	Mode Mode            // reports which strategy was selected
	Pool *pgxpool.Pool   // set when Mode == ModePool
	Conn *pgx.Conn       // set when Mode == ModeConn
}
```

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
| `WithConfig(*manifest.PostgresConfig)` | apply a manifest config section (entry point for config-file setups) |
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

`WithConfig` maps `*manifest.PostgresConfig` field by field: `URI`, `Host`,
`Port` (int), `Username`/`Password`, `DBName`, `SSLMode` → `sslmode`,
`AppName` → `application_name`, `ConnTimeout` → `connect_timeout`, `Mode`
(`"pool"`/`"conn"`), and the `Pool` block. Any of these can be overridden by
a later option.

## Lifecycle

`Close(ctx)` closes the single connection (with error) or the pool.
Implementing `apin.Closer`, it slots directly into `apin.App.Track`.
