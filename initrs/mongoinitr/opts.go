package mongoinitr

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Config contains MongoDB connection settings owned by mongoinitr.
type Config struct {
	URI    string `json:"uri" yaml:"uri" env:"mongodb_uri"`
	DBName string `json:"dbName" yaml:"dbName" env:"mongodb_db_name"`
}

// resolvedConfig is private construction state owned by mongoinitr.
type resolvedConfig struct {
	clientOptions *options.ClientOptions
	dbName        string
	pingTimeout   time.Duration
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithConfig applies an initializer-owned config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *Config) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if config == nil {
			return nil
		}
		return apply(s, []Option{
			WithURI(config.URI),
			WithDBName(config.DBName),
		})
	})
}

// WithURI applies a mongodb connection URI.
func WithURI(uri string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.clientOptions.ApplyURI(uri)
		return nil
	})
}

// WithTimeout sets the driver connect timeout.
func WithTimeout(d time.Duration) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.clientOptions.SetConnectTimeout(d)
		return nil
	})
}

// WithDBName sets the default database of the returned shell.
func WithDBName(name string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.dbName = name
		return nil
	})
}

// WithPingTimeout sets the readiness ping timeout. Defaults to 10s.
func WithPingTimeout(d time.Duration) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.pingTimeout = d
		return nil
	})
}

// WithNativeClientOptions configures driver options not represented by Config
// or the named mongoinitr options.
func WithNativeClientOptions(configure func(*options.ClientOptions) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure != nil {
			return configure(s.clientOptions)
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
			errs = append(errs, fmt.Errorf("mongoinitr: apply option: %w", err))
		}
	}
	return errors.Join(errs...)
}
