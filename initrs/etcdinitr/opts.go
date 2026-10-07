package etcdinitr

import (
	"errors"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Config contains etcd connection settings owned by etcdinitr.
// Endpoints holds one or more etcd endpoints and must not be empty when the
// shell is constructed. Timeout is optional, expressed in seconds, and must be
// positive when set.
type Config struct {
	Endpoints []string `json:"endpoints" yaml:"endpoints" env:"etcd_endpoints"`
	Username  string   `json:"username" yaml:"username" env:"etcd_username"`
	Password  string   `json:"password" yaml:"password" env:"etcd_password"`
	Timeout   *int     `json:"timeout,omitempty" yaml:"timeout,omitempty" env:"etcd_timeout"`
}

// resolvedConfig is private construction state owned by etcdinitr.
type resolvedConfig struct {
	opts       *clientv3.Config
	timeoutErr error
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
		opts := []Option{WithEndpoints(config.Endpoints)}
		if config.Username != "" {
			opts = append(opts, WithUsername(config.Username))
		}
		if config.Password != "" {
			opts = append(opts, WithPassword(config.Password))
		}
		if config.Timeout != nil {
			seconds := *config.Timeout
			if seconds <= 0 {
				s.timeoutErr = fmt.Errorf("etcd timeout must be positive")
			} else if uint64(seconds) > uint64(1<<63-1)/uint64(time.Second) {
				s.timeoutErr = fmt.Errorf("etcd timeout overflows time.Duration")
			} else {
				s.opts.DialTimeout = time.Duration(seconds) * time.Second
				s.timeoutErr = nil
			}
		}
		return apply(s, opts)
	})
}

// WithEndpoints sets the etcd endpoints. The caller's slice is copied before
// it is stored, so later mutations do not affect the shell.
func WithEndpoints(endpoints []string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.Endpoints = append([]string(nil), endpoints...)
		return nil
	})
}

// WithUsername sets the auth username.
func WithUsername(username string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.Username = username
		return nil
	})
}

// WithPassword sets the auth password.
func WithPassword(password string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.Password = password
		return nil
	})
}

// WithTimeout sets the dial timeout.
func WithTimeout(d time.Duration) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.DialTimeout = d
		s.timeoutErr = nil
		return nil
	})
}

// WithNativeConfig configures native etcd client settings not represented by
// Config or the named etcdinitr options.
func WithNativeConfig(configure func(*clientv3.Config) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure != nil {
			if err := configure(s.opts); err != nil {
				return err
			}
			if s.opts.DialTimeout > 0 {
				s.timeoutErr = nil
			}
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
			errs = append(errs, fmt.Errorf("etcdinitr: apply option: %w", err))
		}
	}
	return errors.Join(errs...)
}
