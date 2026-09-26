package etcdinitr

import (
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Config contains etcd connection settings owned by etcdinitr.
// Timeout is optional, expressed in seconds, and must be positive when set.
type Config struct {
	Endpoints string `json:"endpoints" yaml:"endpoints" env:"etcd_endpoints"`
	Username  string `json:"username" yaml:"username" env:"etcd_username"`
	Password  string `json:"password" yaml:"password" env:"etcd_password"`
	Timeout   *int   `json:"timeout,omitempty" yaml:"timeout,omitempty" env:"etcd_timeout"`
}

// Store is the resolved configuration of a shell.
type Store struct {
	Opts *clientv3.Config
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
		if config.Timeout != nil {
			if *config.Timeout <= 0 {
				return fmt.Errorf("etcd timeout must be positive")
			}
			if uint64(*config.Timeout) > uint64(1<<63-1)/uint64(time.Second) {
				return fmt.Errorf("etcd timeout overflows time.Duration")
			}
		}
		opts := []Option{WithEndpoints(strings.Split(config.Endpoints, ","))}
		if config.Username != "" {
			opts = append(opts, WithUsername(config.Username))
		}
		if config.Password != "" {
			opts = append(opts, WithPassword(config.Password))
		}
		if config.Timeout != nil {
			opts = append(opts, WithTimeout(time.Duration(*config.Timeout)*time.Second))
		}
		return apply(s, opts)
	}
}

// WithEndpoints sets the etcd endpoints.
func WithEndpoints(endpoints []string) Option {
	return func(s *Store) error {
		s.Opts.Endpoints = endpoints
		return nil
	}
}

// WithUsername sets the auth username.
func WithUsername(username string) Option {
	return func(s *Store) error {
		s.Opts.Username = username
		return nil
	}
}

// WithPassword sets the auth password.
func WithPassword(password string) Option {
	return func(s *Store) error {
		s.Opts.Password = password
		return nil
	}
}

// WithTimeout sets the dial timeout.
func WithTimeout(d time.Duration) Option {
	return func(s *Store) error {
		s.Opts.DialTimeout = d
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
