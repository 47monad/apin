package rmqinitr

import (
	"time"

	"github.com/47monad/zaal"
	"github.com/go-logr/logr"
)

const (
	defaultMinRetryInterval = time.Second
	defaultMaxRetryInterval = 30 * time.Second
)

// Store is the resolved configuration of a shell.
type Store struct {
	URI              string
	MaxRetryInterval time.Duration
	MinRetryInterval time.Duration
	LazyConnect      bool
	Logger           logr.Logger
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*Store) error

// WithConfig applies a zaal config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *zaal.RabbitMQConfig) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		opts := []Option{WithURI(config.URI)}
		if config.MinRetryInterval > 0 {
			opts = append(opts, WithMinRetryInterval(time.Duration(config.MinRetryInterval)*time.Second))
		}
		if config.MaxRetryInterval > 0 {
			opts = append(opts, WithMaxRetryInterval(time.Duration(config.MaxRetryInterval)*time.Second))
		}
		return apply(s, opts)
	}
}

// WithURI sets the amqp connection URI.
func WithURI(uri string) Option {
	return func(s *Store) error {
		s.URI = uri
		return nil
	}
}

// WithMinRetryInterval sets the initial reconnect backoff. Defaults to 1s.
func WithMinRetryInterval(interval time.Duration) Option {
	return func(s *Store) error {
		s.MinRetryInterval = interval
		return nil
	}
}

// WithMaxRetryInterval sets the reconnect backoff cap. Defaults to 30s.
func WithMaxRetryInterval(interval time.Duration) Option {
	return func(s *Store) error {
		s.MaxRetryInterval = interval
		return nil
	}
}

// WithLazyConnect defers the first connection to the background reconnect
// loop; New then succeeds without reaching the broker. By default New
// connects synchronously and fails on dial errors.
func WithLazyConnect() Option {
	return func(s *Store) error {
		s.LazyConnect = true
		return nil
	}
}

// WithLogger injects a logger for connection lifecycle events. Defaults to
// discarding logs.
func WithLogger(logger logr.Logger) Option {
	return func(s *Store) error {
		s.Logger = logger
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
