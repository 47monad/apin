package etcdinitr

import (
	"strings"
	"time"

	"github.com/47monad/apin/manifest"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Store is the resolved configuration of a shell.
type Store struct {
	Opts *clientv3.Config
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*Store) error

// WithConfig applies a manifest config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *manifest.EtcdConfig) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		opts := []Option{WithEndpoints(strings.Split(config.Endpoints, ","))}
		if config.Username != "" {
			opts = append(opts, WithUsername(config.Username))
		}
		if config.Password != "" {
			opts = append(opts, WithPassword(config.Password))
		}
		if config.Timeout > 0 {
			opts = append(opts, WithTimeout(time.Duration(config.Timeout)*time.Second))
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
