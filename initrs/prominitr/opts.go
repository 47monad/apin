package prominitr

import "errors"

// Config contains Prometheus settings owned by prominitr.
type Config struct {
	GoCollector      bool `json:"goCollector" yaml:"goCollector" env:"prometheus_go_collector"`
	ProcessCollector bool `json:"processCollector" yaml:"processCollector" env:"prometheus_process_collector"`
}

// resolvedConfig is private construction state owned by prominitr.
type resolvedConfig struct {
	goCollector      bool
	processCollector bool
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
		s.goCollector = config.GoCollector
		s.processCollector = config.ProcessCollector
		return nil
	})
}

// WithGoCollector registers the standard Go runtime collectors.
func WithGoCollector(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.goCollector = enabled
		return nil
	})
}

// WithProcessCollector registers the standard process collectors.
func WithProcessCollector(enabled bool) Option {
	return optionFunc(func(s *resolvedConfig) error {
		s.processCollector = enabled
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
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
