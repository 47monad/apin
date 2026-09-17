package mongoinitr

import (
	"time"

	"github.com/47monad/apin/manifest"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Store is the resolved configuration of a shell.
type Store struct {
	Opts        *options.ClientOptions
	DBName      string
	PingTimeout time.Duration
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*Store) error

// WithConfig applies a manifest config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *manifest.MongodbConfig) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		return apply(s, []Option{
			WithURI(config.URI),
			WithDBName(config.DBName),
		})
	}
}

// WithURI applies a mongodb connection URI.
func WithURI(uri string) Option {
	return func(s *Store) error {
		s.Opts.ApplyURI(uri)
		return nil
	}
}

// WithTimeout sets the driver connect timeout.
func WithTimeout(d time.Duration) Option {
	return func(s *Store) error {
		s.Opts.SetConnectTimeout(d)
		return nil
	}
}

// WithDBName sets the default database of the returned shell.
func WithDBName(name string) Option {
	return func(s *Store) error {
		s.DBName = name
		return nil
	}
}

// WithPingTimeout sets the readiness ping timeout. Defaults to 10s.
func WithPingTimeout(d time.Duration) Option {
	return func(s *Store) error {
		s.PingTimeout = d
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
