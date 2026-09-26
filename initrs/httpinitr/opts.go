package httpinitr

import "net/http"

// Store contains resolved HTTP server settings.
type Store struct {
	Port    int
	Handler http.Handler
}

// Option configures an HTTP server. Options are applied in order, so later
// options override earlier values.
type Option func(*Store) error

// WithConfig applies an httpinitr-owned configuration.
func WithConfig(config *Config) Option {
	return func(store *Store) error {
		if config == nil {
			return nil
		}
		return WithPort(config.Port)(store)
	}
}

// WithPort sets the listening port. Zero selects the default port.
func WithPort(port int) Option {
	return func(store *Store) error {
		store.Port = port
		return nil
	}
}

// WithHandler sets the HTTP handler. A nil handler uses a new empty mux.
func WithHandler(handler http.Handler) Option {
	return func(store *Store) error {
		store.Handler = handler
		return nil
	}
}

func apply(store *Store, opts []Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(store); err != nil {
			return err
		}
	}
	return nil
}
