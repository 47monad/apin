package pginitr

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// Config is the PostgreSQL initializer's configuration. Applications may
// embed it in their own aggregate configuration type.
type Config struct {
	URI         string     `json:"uri" yaml:"uri" env:"postgres_uri"`
	Host        string     `json:"host" yaml:"host" env:"postgres_host"`
	Port        int        `json:"port" yaml:"port" env:"postgres_port"`
	Username    string     `json:"username" yaml:"username" env:"postgres_username"`
	Password    string     `json:"password" yaml:"password" env:"postgres_password"`
	DBName      string     `json:"dbName" yaml:"dbName" env:"postgres_db_name"`
	SSLMode     string     `json:"sslMode,omitempty" yaml:"sslMode,omitempty" env:"postgres_ssl_mode"`
	AppName     string     `json:"appName,omitempty" yaml:"appName,omitempty" env:"postgres_app_name"`
	ConnTimeout int        `json:"connTimeout,omitempty" yaml:"connTimeout,omitempty" env:"postgres_conn_timeout"`
	Mode        Mode       `json:"mode,omitempty" yaml:"mode,omitempty" env:"postgres_mode"`
	Pool        PoolConfig `json:"pool,omitempty" yaml:"pool,omitempty"`
}

// DSN returns URI when configured, otherwise it composes a URI from the
// discrete connection fields. The default port is 5432.
func (c *Config) DSN() (string, error) {
	if c.URI != "" {
		return c.URI, nil
	}
	if c.Host == "" {
		return "", errors.New("postgres: host is required when uri is not set")
	}
	if c.DBName == "" {
		return "", errors.New("postgres: dbName is required when uri is not set")
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	u := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", c.Host, port),
		Path:   "/" + c.DBName,
	}
	if c.Username != "" {
		u.User = url.UserPassword(c.Username, c.Password)
	}
	params := url.Values{}
	if c.SSLMode != "" {
		params.Set("sslmode", c.SSLMode)
	}
	if c.AppName != "" {
		params.Set("application_name", c.AppName)
	}
	if c.ConnTimeout > 0 {
		params.Set("connect_timeout", fmt.Sprintf("%d", c.ConnTimeout))
	}
	if len(params) > 0 {
		u.RawQuery = params.Encode()
	}
	return u.String(), nil
}

func validateStore(store *Store) error {
	var errs []error
	if store.Mode != ModePool && store.Mode != ModeConn {
		errs = append(errs, fmt.Errorf("postgres: invalid mode %q (want %q or %q)", store.Mode, ModePool, ModeConn))
	}
	if store.Port != "" {
		port, err := strconv.Atoi(store.Port)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("postgres: port %q out of range (1-65535)", store.Port))
		}
	}
	if uriPort := store.URI.Port(); uriPort != "" {
		port, err := strconv.Atoi(uriPort)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("postgres: URI port %q out of range (1-65535)", uriPort))
		}
	}
	query, err := url.ParseQuery(store.URI.RawQuery)
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
	if store.Pool.MinConns < 0 {
		errs = append(errs, errors.New("postgres: pool.minConns must not be negative"))
	}
	if store.Pool.MaxConns < 0 {
		errs = append(errs, errors.New("postgres: pool.maxConns must not be negative"))
	}
	if store.Pool.MaxConns > 0 && store.Pool.MinConns > store.Pool.MaxConns {
		errs = append(errs, fmt.Errorf("postgres: pool.minConns (%d) must not exceed pool.maxConns (%d)", store.Pool.MinConns, store.Pool.MaxConns))
	}
	for _, field := range []struct {
		name    string
		seconds int
	}{
		{name: "maxConnLifetime", seconds: store.Pool.MaxConnLifetime},
		{name: "maxConnIdleTime", seconds: store.Pool.MaxConnIdleTime},
		{name: "healthCheckInterval", seconds: store.Pool.HealthCheckInterval},
	} {
		if field.seconds < 0 {
			errs = append(errs, fmt.Errorf("postgres: pool.%s must not be negative", field.name))
		}
	}
	return errors.Join(errs...)
}

func validSSLMode(mode string) bool {
	switch mode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}
