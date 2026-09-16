package pginitr

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/47monad/zaal"
)

// Mode selects the connection strategy of the shell.
type Mode string

const (
	// ModePool uses a pgxpool.Pool. This is the default mode.
	ModePool Mode = "pool"
	// ModeConn uses a single *pgx.Conn.
	ModeConn Mode = "conn"
)

// PoolConfig carries pgxpool tuning. Times are in seconds and are only
// applied when the shell runs in ModePool.
type PoolConfig struct {
	MaxConns            int
	MinConns            int
	MaxConnLifetime     int
	MaxConnIdleTime     int
	HealthCheckInterval int
}

// Store is the resolved configuration of a shell. Options are applied to it
// in the order they are passed to New, so later options win.
type Store struct {
	URI  *url.URL
	Port string // composed onto the URI host after all options are applied
	Mode Mode
	Pool PoolConfig
}

// Option mutates the store. Options returning an error fail New immediately.
type Option func(*Store) error

// WithConfig applies a zaal config section. It is the entry point for
// config-file driven setups; later options override individual fields.
func WithConfig(config *zaal.PostgresConfig) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		opts := []Option{
			WithURI(config.URI),
			WithUser(url.UserPassword(config.Username, config.Password)),
			WithHost(config.Host),
			WithPort(config.Port),
			WithDBName(config.DBName),
			WithSSLMode(config.SSLMode),
			WithParam("application_name", config.AppName),
		}
		if config.ConnTimeout > 0 {
			opts = append(opts, WithParam("connect_timeout", strconv.Itoa(config.ConnTimeout)))
		}
		if config.Mode != "" {
			opts = append(opts, WithMode(Mode(config.Mode)))
		}
		opts = append(opts, WithPoolConfig(PoolConfig{
			MaxConns:            config.Pool.MaxConns,
			MinConns:            config.Pool.MinConns,
			MaxConnLifetime:     config.Pool.MaxConnLifetime,
			MaxConnIdleTime:     config.Pool.MaxConnIdleTime,
			HealthCheckInterval: config.Pool.HealthCheckInterval,
		}))
		return apply(s, opts)
	}
}

// WithURI merges connection details from a postgres URI. Query params on the
// URI are preserved unless overridden by later options.
func WithURI(uri string) Option {
	return func(s *Store) error {
		if uri == "" {
			return nil
		}
		parsed, err := ParseURI(uri)
		if err != nil {
			return fmt.Errorf("failed to apply postgres URI: %w", err)
		}
		if parsed.User.Username() != "" {
			s.URI.User = parsed.User
		}
		if parsed.Path != "" {
			s.URI.Path = parsed.Path
		}
		if parsed.Host != "" {
			s.URI.Host = parsed.Host
		}
		if parsed.RawQuery != "" {
			query, err := url.ParseQuery(parsed.RawQuery)
			if err != nil {
				return fmt.Errorf("failed to apply postgres URI: %w", err)
			}
			existing, err := url.ParseQuery(s.URI.RawQuery)
			if err != nil {
				return fmt.Errorf("failed to apply postgres URI: %w", err)
			}
			for key, values := range query {
				if _, ok := existing[key]; !ok {
					existing[key] = values
				}
			}
			s.URI.RawQuery = existing.Encode()
		}
		return nil
	}
}

// WithUser sets explicit URL user info, taking precedence over URI-derived
// credentials.
func WithUser(user *url.Userinfo) Option {
	return func(s *Store) error {
		s.URI.User = user
		return nil
	}
}

// WithHost sets the database host.
func WithHost(host string) Option {
	return func(s *Store) error {
		s.URI.Host = host
		return nil
	}
}

// WithPort sets the database port. It composes with WithHost and URIs
// regardless of option order.
func WithPort(port int) Option {
	return func(s *Store) error {
		if port != 0 {
			s.Port = strconv.Itoa(port)
		}
		return nil
	}
}

// WithDBName sets the database name.
func WithDBName(dbname string) Option {
	return func(s *Store) error {
		if dbname != "" {
			s.URI.Path = dbname
		}
		return nil
	}
}

// WithSSLMode sets the sslmode URI parameter.
func WithSSLMode(sslmode string) Option {
	return WithParam("sslmode", sslmode)
}

// WithParam sets a single URI parameter, overriding same-named parameters
// from earlier options.
func WithParam(key, value string) Option {
	return func(s *Store) error {
		if value == "" {
			return nil
		}
		query, err := url.ParseQuery(s.URI.RawQuery)
		if err != nil {
			return fmt.Errorf("failed to parse postgres URI query: %w", err)
		}
		query.Set(key, value)
		s.URI.RawQuery = query.Encode()
		return nil
	}
}

// WithMode sets the connection strategy explicitly.
func WithMode(mode Mode) Option {
	return func(s *Store) error {
		switch mode {
		case ModePool, ModeConn:
			s.Mode = mode
			return nil
		default:
			return fmt.Errorf("invalid pginitr mode: %q", mode)
		}
	}
}

// WithPool selects pool mode. Pool mode is the default.
func WithPool() Option {
	return WithMode(ModePool)
}

// WithSingleConn selects single-connection mode.
func WithSingleConn() Option {
	return WithMode(ModeConn)
}

// WithPoolConfig sets pool tuning. Only applied in pool mode.
func WithPoolConfig(pool PoolConfig) Option {
	return func(s *Store) error {
		s.Pool = pool
		return nil
	}
}

func apply(s *Store, opts []Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(s); err != nil {
			return err
		}
	}
	return nil
}
