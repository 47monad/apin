package httpinitr

import (
	"errors"
	"net/http"
)

// resolvedConfig is private construction state owned by httpinitr.
type resolvedConfig struct {
	port    int
	handler http.Handler
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithConfig applies an httpinitr-owned configuration.
func WithConfig(config *Config) Option {
	return optionFunc(func(store *resolvedConfig) error {
		if config == nil {
			return nil
		}
		return WithPort(config.Port).apply(store)
	})
}

// WithPort sets the listening port. Zero selects the default port.
func WithPort(port int) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.port = port
		return nil
	})
}

// WithHandler sets the HTTP handler. A nil handler uses a new empty mux.
func WithHandler(handler http.Handler) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.handler = handler
		return nil
	})
}

func apply(store *resolvedConfig, opts []Option) error {
	var errs []error
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(store); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
