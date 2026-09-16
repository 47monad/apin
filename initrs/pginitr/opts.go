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

type Store struct {
	URI  *url.URL
	Mode Mode
	Pool PoolConfig
}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	store := &Store{
		URI: &url.URL{Scheme: "postgres"},
	}

	for _, opt := range b.Opts {
		if opt == nil {
			continue
		}

		if err := opt(store); err != nil {
			return nil, err
		}
	}

	if store.Mode == "" {
		store.Mode = ModePool
	}

	return store, nil
}

func (b *Builder) WithConfig(config *zaal.PostgresConfig) *Builder {
	if config.URI != "" {
		b.ApplyURI(config.URI)
	}
	if config.Username != "" {
		b.SetUser(url.UserPassword(config.Username, config.Password))
	}
	if config.Host != "" {
		b.SetHost(config.Host)
	}
	if config.Port != 0 {
		b.SetPort(strconv.Itoa(config.Port))
	}
	if config.DBName != "" {
		b.SetDBName(config.DBName)
	}
	if config.SSLMode != "" {
		b.SetSSLMode(config.SSLMode)
	}
	if config.AppName != "" {
		b.SetParam("application_name", config.AppName)
	}
	if config.ConnTimeout > 0 {
		b.SetParam("connect_timeout", strconv.Itoa(config.ConnTimeout))
	}
	if config.Mode != "" {
		b.WithMode(Mode(config.Mode))
	}
	b.SetPoolConfig(PoolConfig{
		MaxConns:            config.Pool.MaxConns,
		MinConns:            config.Pool.MinConns,
		MaxConnLifetime:     config.Pool.MaxConnLifetime,
		MaxConnIdleTime:     config.Pool.MaxConnIdleTime,
		HealthCheckInterval: config.Pool.HealthCheckInterval,
	})
	return b
}

func (b *Builder) WithMode(mode Mode) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		switch mode {
		case ModePool, ModeConn:
			s.Mode = mode
			return nil
		default:
			return fmt.Errorf("invalid pginitr mode: %q", mode)
		}
	})
	return b
}

func (b *Builder) WithPool() *Builder {
	return b.WithMode(ModePool)
}

func (b *Builder) WithSingleConn() *Builder {
	return b.WithMode(ModeConn)
}

func (b *Builder) ApplyURI(uri string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
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
	})
	return b
}

func (b *Builder) SetUser(user *url.Userinfo) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI.User = user
		return nil
	})
	return b
}

func (b *Builder) SetHost(host string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI.Host = host
		return nil
	})
	return b
}

func (b *Builder) SetPort(port string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		host := s.URI.Hostname()
		if host == "" {
			return nil
		}
		s.URI.Host = host + ":" + port
		return nil
	})
	return b
}

func (b *Builder) SetDBName(dbname string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI.Path = dbname
		return nil
	})
	return b
}

func (b *Builder) SetSSLMode(sslmode string) *Builder {
	return b.SetParam("sslmode", sslmode)
}

func (b *Builder) SetParam(key, value string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		if s.URI.RawQuery == "" {
			s.URI.RawQuery = url.Values{key: []string{value}}.Encode()
			return nil
		}
		query, err := url.ParseQuery(s.URI.RawQuery)
		if err != nil {
			return fmt.Errorf("failed to parse postgres URI query: %w", err)
		}
		query.Set(key, value)
		s.URI.RawQuery = query.Encode()
		return nil
	})
	return b
}

func (b *Builder) SetPoolConfig(pool PoolConfig) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.Pool = pool
		return nil
	})
	return b
}

func Opts() *Builder {
	return &Builder{}
}
