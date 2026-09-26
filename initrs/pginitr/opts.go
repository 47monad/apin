package pginitr

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	MaxConns            int `json:"maxConns,omitempty" yaml:"maxConns,omitempty" env:"postgres_pool_max_conns"`
	MinConns            int `json:"minConns,omitempty" yaml:"minConns,omitempty" env:"postgres_pool_min_conns"`
	MaxConnLifetime     int `json:"maxConnLifetime,omitempty" yaml:"maxConnLifetime,omitempty" env:"postgres_pool_max_conn_lifetime"`
	MaxConnIdleTime     int `json:"maxConnIdleTime,omitempty" yaml:"maxConnIdleTime,omitempty" env:"postgres_pool_max_conn_idle_time"`
	HealthCheckInterval int `json:"healthCheckInterval,omitempty" yaml:"healthCheckInterval,omitempty" env:"postgres_pool_health_check_interval"`
}

// resolvedConfig is the private construction state. Options are applied in
// order, so later options win.
type resolvedConfig struct {
	uri         *url.URL
	port        string // composed onto the URI host after all options are applied
	mode        Mode
	pool        PoolConfig
	poolOptions []poolConfigOption
	connOptions []func(*pgx.ConnConfig) error
}

type poolConfigOption struct {
	tuning    *PoolConfig
	configure func(*pgxpool.Config) error
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithConfig applies an initializer-owned config; later options override
// individual fields.
func WithConfig(config *Config) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if config == nil {
			return nil
		}
		opts := []Option{WithURI(config.URI)}
		if config.Username != "" || config.Password != "" {
			opts = append(opts, WithUser(url.UserPassword(config.Username, config.Password)))
		}
		opts = append(opts,
			WithHost(config.Host),
			WithPort(config.Port),
			WithDBName(config.DBName),
			WithSSLMode(config.SSLMode),
			WithParam("application_name", config.AppName),
		)
		if config.ConnTimeout != 0 {
			opts = append(opts, WithParam("connect_timeout", strconv.Itoa(config.ConnTimeout)))
		}
		if config.Mode != "" {
			opts = append(opts, WithMode(Mode(config.Mode)))
		}
		opts = append(opts, WithPoolConfig(config.Pool))
		return apply(s, opts)
	})
}

// WithURI merges connection details from a postgres URI. Query params on the
// URI are preserved unless overridden by later options.
func WithURI(uri string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if uri == "" {
			return nil
		}
		parsed, err := ParseURI(uri)
		if err != nil {
			return fmt.Errorf("failed to apply postgres URI: %w", err)
		}
		if parsed.User.Username() != "" {
			s.uri.User = parsed.User
		}
		if parsed.Path != "" {
			s.uri.Path = parsed.Path
		}
		if parsed.Host != "" {
			s.uri.Host = parsed.Host
		}
		if parsed.RawQuery != "" {
			query, err := url.ParseQuery(parsed.RawQuery)
			if err != nil {
				return fmt.Errorf("failed to apply postgres URI: %w", err)
			}
			existing, err := url.ParseQuery(s.uri.RawQuery)
			if err != nil {
				return fmt.Errorf("failed to apply postgres URI: %w", err)
			}
			for key, values := range query {
				if _, ok := existing[key]; !ok {
					existing[key] = values
				}
			}
			s.uri.RawQuery = existing.Encode()
		}
		return nil
	})
}

// WithUser sets explicit URL user info, taking precedence over URI-derived
// credentials.
func WithUser(user *url.Userinfo) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.uri.User = user
		return nil
	})
}

// WithHost sets the database host.
func WithHost(host string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.uri.Host = host
		return nil
	})
}

// WithPort sets the database port. It composes with WithHost and URIs
// regardless of option order.
func WithPort(port int) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if port != 0 {
			s.port = strconv.Itoa(port)
		}
		return nil
	})
}

// WithDBName sets the database name.
func WithDBName(dbname string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if dbname != "" {
			s.uri.Path = dbname
		}
		return nil
	})
}

// WithSSLMode sets the sslmode URI parameter.
func WithSSLMode(sslmode string) Option {
	return WithParam("sslmode", sslmode)
}

// WithParam sets a single URI parameter, overriding same-named parameters
// from earlier options.
func WithParam(key, value string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if value == "" {
			return nil
		}
		query, err := url.ParseQuery(s.uri.RawQuery)
		if err != nil {
			return fmt.Errorf("failed to parse postgres URI query: %w", err)
		}
		query.Set(key, value)
		s.uri.RawQuery = query.Encode()
		return nil
	})
}

// WithMode sets the connection strategy explicitly.
func WithMode(mode Mode) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.mode = mode
		return nil
	})
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
	return optionFunc(func(s *resolvedConfig) error {
		s.pool = pool
		poolCopy := pool
		s.poolOptions = append(s.poolOptions, poolConfigOption{tuning: &poolCopy})
		return nil
	})
}

// WithNativePoolConfig adds a deliberate escape hatch for pgxpool settings
// not represented by pginitr.PoolConfig. It applies in pool mode.
func WithNativePoolConfig(configure func(*pgxpool.Config) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure != nil {
			s.poolOptions = append(s.poolOptions, poolConfigOption{configure: configure})
		}
		return nil
	})
}

// WithNativeConnConfig adds a deliberate escape hatch for pgx connection
// settings not represented by pginitr.Config. It applies in either mode.
func WithNativeConnConfig(configure func(*pgx.ConnConfig) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure != nil {
			s.connOptions = append(s.connOptions, configure)
			s.poolOptions = append(s.poolOptions, poolConfigOption{configure: func(config *pgxpool.Config) error {
				return configure(config.ConnConfig)
			}})
		}
		return nil
	})
}

func apply(s *resolvedConfig, opts []Option) error {
	var errs []error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(s); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
