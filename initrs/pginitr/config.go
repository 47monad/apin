package pginitr

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
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
// discrete connection fields. The default port is 5432. An IPv6 host is
// bracketed exactly once, whether it is written bare (fd00::1) or already
// bracketed ([fd00::1]).
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
		Host:   net.JoinHostPort(hostForJoin(c.Host), strconv.Itoa(port)),
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

// hostForJoin strips the brackets from an IPv6 literal, so net.JoinHostPort is
// the single place that brackets the host. Config accepts either form, which
// keeps a host copied out of a connection URI working instead of producing
// doubled brackets.
func hostForJoin(host string) string {
	return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
}

func validateConfig(config *resolvedConfig) error {
	var errs []error
	if config.mode != ModePool && config.mode != ModeConn {
		errs = append(errs, fmt.Errorf("postgres: invalid mode %q (want %q or %q)", config.mode, ModePool, ModeConn))
	}
	if config.port != "" {
		port, err := strconv.Atoi(config.port)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("postgres: port %q out of range (1-65535)", config.port))
		}
	}
	if uriPort := config.uri.Port(); uriPort != "" {
		port, err := strconv.Atoi(uriPort)
		if err != nil || port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("postgres: URI port %q out of range (1-65535)", uriPort))
		}
	}
	query, err := url.ParseQuery(config.uri.RawQuery)
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
	if config.pool.MinConns < 0 {
		errs = append(errs, errors.New("postgres: pool.minConns must not be negative"))
	}
	if config.pool.MaxConns < 0 {
		errs = append(errs, errors.New("postgres: pool.maxConns must not be negative"))
	}
	if config.pool.MaxConns > 0 && config.pool.MinConns > config.pool.MaxConns {
		errs = append(errs, fmt.Errorf("postgres: pool.minConns (%d) must not exceed pool.maxConns (%d)", config.pool.MinConns, config.pool.MaxConns))
	}
	for _, field := range []struct {
		name    string
		seconds int
	}{
		{name: "maxConnLifetime", seconds: config.pool.MaxConnLifetime},
		{name: "maxConnIdleTime", seconds: config.pool.MaxConnIdleTime},
		{name: "healthCheckInterval", seconds: config.pool.HealthCheckInterval},
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
