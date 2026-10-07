package pginitr

import "errors"

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
// discrete connection fields through the same resolver New uses. The default
// port is 5432. An IPv6 host is bracketed exactly once, whether it is written
// bare (fd00::1) or already bracketed ([fd00::1]).
func (c *Config) DSN() (string, error) {
	if c.URI != "" {
		return c.URI, nil
	}

	r := newResolver()
	if err := r.apply([]Option{WithConfig(c)}); err != nil {
		return "", err
	}
	if r.uri.Host == "" {
		return "", errors.New("postgres: host is required when uri is not set")
	}
	if r.uri.Path == "" {
		return "", errors.New("postgres: dbName is required when uri is not set")
	}
	return r.compose().String(), nil
}

func validSSLMode(mode string) bool {
	switch mode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}
