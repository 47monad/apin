package pginitr

import (
	"fmt"
	"net/url"
	"strings"
)

// ParsePostgresURI parses a PostgreSQL connection URI and returns a PostgresConfig struct
func ParseURI(uri string) (*url.URL, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URI: %w", err)
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, fmt.Errorf("invalid scheme: expected 'postgres' or 'postgresql', got '%s'", u.Scheme)
	}

	if u.Path != "" {
		u.Path = strings.TrimPrefix(u.Path, "/")
	}

	return u, nil
}
