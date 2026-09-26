package rmqinitr

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-logr/logr"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultMinRetryInterval = time.Second
	defaultMaxRetryInterval = 30 * time.Second
)

// Config contains RabbitMQ connection settings owned by rmqinitr. Retry
// intervals are optional, expressed in seconds, and validated after options
// have been applied.
type Config struct {
	URI              string `json:"uri" yaml:"uri" env:"rabbitmq_uri"`
	MinRetryInterval *int   `json:"minRetryInterval,omitempty" yaml:"minRetryInterval,omitempty" env:"rabbitmq_min_retry_interval"`
	MaxRetryInterval *int   `json:"maxRetryInterval,omitempty" yaml:"maxRetryInterval,omitempty" env:"rabbitmq_max_retry_interval"`
}

// resolvedConfig is private construction state owned by rmqinitr.
type resolvedConfig struct {
	uri              string
	maxRetryInterval time.Duration
	minRetryInterval time.Duration
	lazyConnect      bool
	logger           logr.Logger
	dialConfig       *amqp.Config
	minRetryErr      error
	maxRetryErr      error
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
		opts := []Option{WithURI(config.URI)}
		if config.MinRetryInterval != nil {
			s.minRetryErr = nil
			if *config.MinRetryInterval <= 0 {
				s.minRetryErr = fmt.Errorf("rmqinitr: min retry interval must be positive")
			} else if duration, err := secondsDuration(*config.MinRetryInterval); err != nil {
				s.minRetryErr = fmt.Errorf("rmqinitr: min retry interval: %w", err)
			} else {
				opts = append(opts, WithMinRetryInterval(duration))
			}
		}
		if config.MaxRetryInterval != nil {
			s.maxRetryErr = nil
			if *config.MaxRetryInterval <= 1 {
				s.maxRetryErr = fmt.Errorf("rmqinitr: max retry interval must be greater than 1 second")
			} else if duration, err := secondsDuration(*config.MaxRetryInterval); err != nil {
				s.maxRetryErr = fmt.Errorf("rmqinitr: max retry interval: %w", err)
			} else {
				opts = append(opts, WithMaxRetryInterval(duration))
			}
		}
		return apply(s, opts)
	})
}

func secondsDuration(seconds int) (time.Duration, error) {
	if uint64(seconds) > uint64(math.MaxInt64)/uint64(time.Second) {
		return 0, fmt.Errorf("%d seconds overflows time.Duration", seconds)
	}
	return time.Duration(seconds) * time.Second, nil
}

// WithURI sets the amqp connection URI.
func WithURI(uri string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.uri = uri
		return nil
	})
}

// WithMinRetryInterval sets the initial reconnect backoff. Defaults to 1s.
func WithMinRetryInterval(interval time.Duration) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.minRetryInterval = interval
		s.minRetryErr = nil
		return nil
	})
}

// WithMaxRetryInterval sets the reconnect backoff cap. Defaults to 30s.
func WithMaxRetryInterval(interval time.Duration) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.maxRetryInterval = interval
		s.maxRetryErr = nil
		return nil
	})
}

// WithLazyConnect defers the first connection to the background reconnect
// loop; New then succeeds without reaching the broker. By default New
// connects synchronously and fails on dial errors.
func WithLazyConnect() Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.lazyConnect = true
		return nil
	})
}

// WithLogger injects a logger for connection lifecycle events. Defaults to
// discarding logs.
func WithLogger(logger logr.Logger) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.logger = logger
		return nil
	})
}

// WithNativeDialConfig configures AMQP transport and handshake settings not
// represented by Config or the named rmqinitr options.
func WithNativeDialConfig(configure func(*amqp.Config) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure == nil {
			return nil
		}
		if s.dialConfig == nil {
			s.dialConfig = &amqp.Config{Locale: "en_US"}
		}
		return configure(s.dialConfig)
	})
}

func apply(s *resolvedConfig, opts []Option) error {
	var errs []error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(s); err != nil {
			errs = append(errs, fmt.Errorf("rmqinitr: apply option: %w", err))
		}
	}
	return errors.Join(errs...)
}
