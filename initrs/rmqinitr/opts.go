package rmqinitr

import (
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
)

const (
	defaultMinRetryInterval = time.Second
	defaultMaxRetryInterval = 30 * time.Second
)

// Config contains RabbitMQ connection settings owned by rmqinitr. Retry
// intervals are optional, expressed in seconds, and validated when set.
type Config struct {
	URI              string `json:"uri" yaml:"uri" env:"rabbitmq_uri"`
	MinRetryInterval *int   `json:"minRetryInterval,omitempty" yaml:"minRetryInterval,omitempty" env:"rabbitmq_min_retry_interval"`
	MaxRetryInterval *int   `json:"maxRetryInterval,omitempty" yaml:"maxRetryInterval,omitempty" env:"rabbitmq_max_retry_interval"`
}

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

// WithConfig applies an initializer-owned config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *Config) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		opts := []Option{WithURI(config.URI)}
		if config.MinRetryInterval != nil {
			if *config.MinRetryInterval <= 0 {
				return fmt.Errorf("rmqinitr: min retry interval must be positive")
			}
			duration, err := secondsDuration(*config.MinRetryInterval)
			if err != nil {
				return fmt.Errorf("rmqinitr: min retry interval: %w", err)
			}
			opts = append(opts, WithMinRetryInterval(duration))
		}
		if config.MaxRetryInterval != nil {
			if *config.MaxRetryInterval <= 1 {
				return fmt.Errorf("rmqinitr: max retry interval must be greater than 1 second")
			}
			duration, err := secondsDuration(*config.MaxRetryInterval)
			if err != nil {
				return fmt.Errorf("rmqinitr: max retry interval: %w", err)
			}
			opts = append(opts, WithMaxRetryInterval(duration))
		}
		return apply(s, opts)
	}
}

func secondsDuration(seconds int) (time.Duration, error) {
	if uint64(seconds) > uint64(math.MaxInt64)/uint64(time.Second) {
		return 0, fmt.Errorf("%d seconds overflows time.Duration", seconds)
	}
	return time.Duration(seconds) * time.Second, nil
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
