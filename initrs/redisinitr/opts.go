package redisinitr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Config contains Redis connection settings owned by redisinitr. Addresses
// lists one or more endpoints: a single endpoint uses a standalone client,
// several use a cluster client, and MasterName selects Sentinel. Database
// applies to standalone and Sentinel deployments.
type Config struct {
	Addresses  []string `json:"addresses" yaml:"addresses" env:"redis_addresses"`
	Username   string   `json:"username,omitempty" yaml:"username,omitempty" env:"redis_username"`
	Password   string   `json:"password,omitempty" yaml:"password,omitempty" env:"redis_password"`
	Database   int      `json:"database,omitempty" yaml:"database,omitempty" env:"redis_database"`
	MasterName string   `json:"masterName,omitempty" yaml:"masterName,omitempty" env:"redis_master_name"`
}

// resolvedConfig is private construction state owned by redisinitr.
type resolvedConfig struct {
	opts *redis.UniversalOptions
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithConfig applies an initializer-owned config section. It is the entry point
// for config-file driven setups.
func WithConfig(config *Config) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if config == nil {
			return nil
		}
		opts := []Option{
			WithAddresses(config.Addresses),
			WithUsername(config.Username),
			WithPassword(config.Password),
			WithDatabase(config.Database),
		}
		if config.MasterName != "" {
			opts = append(opts, WithMasterName(config.MasterName))
		}
		return apply(s, opts)
	})
}

// WithAddresses sets the Redis endpoints. The caller's slice is copied before
// it is stored, so later mutations do not affect the shell.
func WithAddresses(addresses []string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.Addrs = append([]string(nil), addresses...)
		return nil
	})
}

// WithUsername sets the ACL username.
func WithUsername(username string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.Username = username
		return nil
	})
}

// WithPassword sets the password.
func WithPassword(password string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.Password = password
		return nil
	})
}

// WithDatabase selects the logical database for standalone and Sentinel
// deployments.
func WithDatabase(database int) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.DB = database
		return nil
	})
}

// WithMasterName selects Sentinel mode and sets the monitored master name.
func WithMasterName(master string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.opts.MasterName = master
		return nil
	})
}

// WithNativeOptions configures native go-redis settings not represented by
// Config or the named options.
func WithNativeOptions(configure func(*redis.UniversalOptions) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure == nil {
			return nil
		}
		return configure(s.opts)
	})
}

func apply(s *resolvedConfig, opts []Option) error {
	var errs []error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(s); err != nil {
			errs = append(errs, fmt.Errorf("redisinitr: apply option: %w", err))
		}
	}
	return errors.Join(errs...)
}

func validateConfig(config *resolvedConfig) error {
	if len(config.opts.Addrs) == 0 {
		return errors.New("redisinitr: at least one address is required")
	}
	for i, address := range config.opts.Addrs {
		if strings.TrimSpace(address) == "" {
			return fmt.Errorf("redisinitr: address %d must not be blank", i)
		}
	}
	if config.opts.DB < 0 {
		return errors.New("redisinitr: database must not be negative")
	}
	return nil
}
