package pginitr

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

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

// defaultPort is applied by the resolver when no port is configured, matching
// PostgreSQL's default.
const defaultPort = "5432"

// PoolConfig carries pgxpool tuning. Times are in seconds and are only
// applied when the shell runs in ModePool.
type PoolConfig struct {
	MaxConns            int `json:"maxConns,omitempty" yaml:"maxConns,omitempty" env:"postgres_pool_max_conns"`
	MinConns            int `json:"minConns,omitempty" yaml:"minConns,omitempty" env:"postgres_pool_min_conns"`
	MaxConnLifetime     int `json:"maxConnLifetime,omitempty" yaml:"maxConnLifetime,omitempty" env:"postgres_pool_max_conn_lifetime"`
	MaxConnIdleTime     int `json:"maxConnIdleTime,omitempty" yaml:"maxConnIdleTime,omitempty" env:"postgres_pool_max_conn_idle_time"`
	HealthCheckInterval int `json:"healthCheckInterval,omitempty" yaml:"healthCheckInterval,omitempty" env:"postgres_pool_health_check_interval"`
}

// resolver turns options into a composed connection URI and validates the
// accumulated configuration. It is the private construction state; options are
// applied in order, so later options win.
type resolver struct {
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
	apply(*resolver) error
}

type optionFunc func(*resolver) error

func (option optionFunc) apply(config *resolver) error {
	return option(config)
}

// WithConfig applies an initializer-owned config; later options override
// individual fields.
func WithConfig(config *Config) Option {
	return optionFunc(func(s *resolver) error {
		if config == nil {
			return nil
		}
		opts := []Option{WithURI(config.URI)}
		if config.Username != "" || config.Password != "" {
			opts = append(opts, WithUser(url.UserPassword(config.Username, config.Password)))
		}
		if config.Host != "" {
			opts = append(opts, WithHost(config.Host))
		}
		opts = append(opts,
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
		return s.apply(opts)
	})
}

// WithURI merges connection details from a postgres URI. Query params on the
// URI replace same-named params set by earlier options, matching the
// later-option-wins contract; params the URI omits are preserved.
func WithURI(uri string) Option {
	return optionFunc(func(s *resolver) error {
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
			// Later options win: a same-named parameter on this URI replaces
			// the one accumulated from earlier options. Assigning the whole
			// slice keeps a URI's repeated parameters intact.
			for key, values := range query {
				existing[key] = values
			}
			s.uri.RawQuery = existing.Encode()
		}
		return nil
	})
}

// WithUser sets explicit URL user info, taking precedence over URI-derived
// credentials.
func WithUser(user *url.Userinfo) Option {
	return optionFunc(func(s *resolver) error {
		s.uri.User = user
		return nil
	})
}

// WithHost sets the database host.
func WithHost(host string) Option {
	return optionFunc(func(s *resolver) error {
		s.uri.Host = host
		return nil
	})
}

// WithPort sets the database port. It composes with WithHost and URIs
// regardless of option order.
func WithPort(port int) Option {
	return optionFunc(func(s *resolver) error {
		if port != 0 {
			s.port = strconv.Itoa(port)
		}
		return nil
	})
}

// WithDBName sets the database name.
func WithDBName(dbname string) Option {
	return optionFunc(func(s *resolver) error {
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
	return optionFunc(func(s *resolver) error {
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
	return optionFunc(func(s *resolver) error {
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
	return optionFunc(func(s *resolver) error {
		s.pool = pool
		poolCopy := pool
		s.poolOptions = append(s.poolOptions, poolConfigOption{tuning: &poolCopy})
		return nil
	})
}

// WithNativePoolConfig adds a deliberate escape hatch for pgxpool settings
// not represented by pginitr.PoolConfig. It applies in pool mode.
func WithNativePoolConfig(configure func(*pgxpool.Config) error) Option {
	return optionFunc(func(s *resolver) error {
		if configure != nil {
			s.poolOptions = append(s.poolOptions, poolConfigOption{configure: configure})
		}
		return nil
	})
}

// WithNativeConnConfig adds a deliberate escape hatch for pgx connection
// settings not represented by pginitr.Config. It applies in either mode.
func WithNativeConnConfig(configure func(*pgx.ConnConfig) error) Option {
	return optionFunc(func(s *resolver) error {
		if configure != nil {
			s.connOptions = append(s.connOptions, configure)
			s.poolOptions = append(s.poolOptions, poolConfigOption{configure: func(config *pgxpool.Config) error {
				return configure(config.ConnConfig)
			}})
		}
		return nil
	})
}

// newResolver returns the construction state with the defaults New and
// Config.DSN share.
func newResolver() *resolver {
	return &resolver{
		uri:  &url.URL{Scheme: "postgres"},
		mode: ModePool,
	}
}

// apply runs options in order; later options win.
func (r *resolver) apply(opts []Option) error {
	var errs []error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(r); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// resolve applies options and validates the accumulated configuration. It is
// the single entry point New uses to turn options into connection settings.
func resolve(opts []Option) (*resolver, error) {
	r := newResolver()
	if err := r.apply(opts); err != nil {
		return nil, err
	}
	if r.uri.User == nil && r.uri.Host == "" && r.uri.Path == "" {
		return nil, fmt.Errorf("pginitr: no postgres configuration provided; pass WithConfig, WithURI, or connection options such as WithHost/WithDBName")
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// compose returns the effective connection URI. The pending port is composed
// onto the host after all options are applied, and a host without a port gets
// the default port, so option order does not matter. New and Config.DSN share
// it, which keeps the two public entry points from drifting.
func (r *resolver) compose() *url.URL {
	uri := *r.uri
	host, port := splitHostPort(uri.Host)
	if host == "" {
		return &uri
	}
	if r.port != "" {
		port = r.port
	}
	if port == "" {
		port = defaultPort
	}
	uri.Host = net.JoinHostPort(host, port)
	return &uri
}

// splitHostPort separates a configured host into its host and port parts. It
// accepts a bare IPv6 literal (fd00::1) that url.Hostname would otherwise
// mistake for host:port because of the embedded colons.
func splitHostPort(hostish string) (host, port string) {
	if strings.HasPrefix(hostish, "[") {
		if host, port, err := net.SplitHostPort(hostish); err == nil {
			return host, port
		}
		return strings.Trim(hostish, "[]"), ""
	}
	if strings.Count(hostish, ":") > 1 {
		return hostish, ""
	}
	if host, port, err := net.SplitHostPort(hostish); err == nil {
		return host, port
	}
	return hostish, ""
}

// validate checks the accumulated configuration.
func (r *resolver) validate() error {
	var errs []error
	if r.mode != ModePool && r.mode != ModeConn {
		errs = append(errs, fmt.Errorf("postgres: invalid mode %q (want %q or %q)", r.mode, ModePool, ModeConn))
	}
	if r.port != "" {
		port, err := strconv.Atoi(r.port)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("postgres: port %q out of range (1-65535)", r.port))
		}
	}
	if _, uriPort := splitHostPort(r.uri.Host); uriPort != "" {
		port, err := strconv.Atoi(uriPort)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("postgres: URI port %q out of range (1-65535)", uriPort))
		}
	}
	query, err := url.ParseQuery(r.uri.RawQuery)
	if err != nil {
		return errors.Join(append(errs, fmt.Errorf("postgres: invalid connection query: %w", err))...)
	}
	if sslmode := query.Get("sslmode"); sslmode != "" && !validSSLMode(sslmode) {
		errs = append(errs, fmt.Errorf("postgres: invalid sslMode %q", sslmode))
	}
	if timeout := query.Get("connect_timeout"); timeout != "" {
		seconds, err := strconv.Atoi(timeout)
		if err != nil || seconds < 0 {
			errs = append(errs, errors.New("postgres: connTimeout must not be negative or invalid"))
		}
	}
	if r.pool.MinConns < 0 {
		errs = append(errs, errors.New("postgres: pool.minConns must not be negative"))
	}
	if r.pool.MaxConns < 0 {
		errs = append(errs, errors.New("postgres: pool.maxConns must not be negative"))
	}
	if r.pool.MaxConns > 0 && r.pool.MinConns > r.pool.MaxConns {
		errs = append(errs, fmt.Errorf("postgres: pool.minConns (%d) must not exceed pool.maxConns (%d)", r.pool.MinConns, r.pool.MaxConns))
	}
	for _, field := range []struct {
		name    string
		seconds int
	}{
		{name: "maxConnLifetime", seconds: r.pool.MaxConnLifetime},
		{name: "maxConnIdleTime", seconds: r.pool.MaxConnIdleTime},
		{name: "healthCheckInterval", seconds: r.pool.HealthCheckInterval},
	} {
		if field.seconds < 0 {
			errs = append(errs, fmt.Errorf("postgres: pool.%s must not be negative", field.name))
		}
	}
	return errors.Join(errs...)
}
