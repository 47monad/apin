package manifest

import (
	"errors"
	"fmt"
	"net/url"
)

// Postgres connection modes.
const (
	PostgresModePool   = "pool"
	PostgresModeSingle = "single"
)

// Valid values for PostgresConfig.SSLMode, matching libpq/Postgres.
var postgresSSLModes = []string{
	"disable",
	"allow",
	"prefer",
	"require",
	"verify-ca",
	"verify-full",
}

// DSN returns a PostgreSQL connection string for the config.
//
// If URI is set it is returned as-is and takes precedence over the
// decomposed fields. Otherwise a DSN is composed from Host, Port,
// Username, Password and DBName, returning an error when a required
// field is missing.
func (c *PostgresConfig) DSN() (string, error) {
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

// Validate checks the config for values that the CUE schema and the
// environment-variable loader cannot enforce, such as enum members and
// cross-field relationships. It returns all violations joined into a
// single error, or nil when the config is valid.
func (c *PostgresConfig) Validate() error {
	var errs []error

	switch c.Mode {
	case "", PostgresModePool, PostgresModeSingle:
	default:
		errs = append(errs, fmt.Errorf("postgres: invalid mode %q (want %q or %q)",
			c.Mode, PostgresModePool, PostgresModeSingle))
	}

	if c.SSLMode != "" && !contains(postgresSSLModes, c.SSLMode) {
		errs = append(errs, fmt.Errorf("postgres: invalid sslMode %q (want one of %v)",
			c.SSLMode, postgresSSLModes))
	}

	if c.Port < 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("postgres: port %d out of range (0-65535)", c.Port))
	}

	if c.ConnTimeout < 0 {
		errs = append(errs, fmt.Errorf("postgres: connTimeout must not be negative"))
	}

	if c.Pool.MinConns < 0 {
		errs = append(errs, fmt.Errorf("postgres: pool.minConns must not be negative"))
	}
	if c.Pool.MaxConns < 0 {
		errs = append(errs, fmt.Errorf("postgres: pool.maxConns must not be negative"))
	}
	if c.Pool.MaxConns > 0 && c.Pool.MinConns > c.Pool.MaxConns {
		errs = append(errs, fmt.Errorf("postgres: pool.minConns (%d) must not exceed pool.maxConns (%d)",
			c.Pool.MinConns, c.Pool.MaxConns))
	}
	for name, secs := range map[string]int{
		"maxConnLifetime":     c.Pool.MaxConnLifetime,
		"maxConnIdleTime":     c.Pool.MaxConnIdleTime,
		"healthCheckInterval": c.Pool.HealthCheckInterval,
	} {
		if secs < 0 {
			errs = append(errs, fmt.Errorf("postgres: pool.%s must not be negative", name))
		}
	}

	return errors.Join(errs...)
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}
