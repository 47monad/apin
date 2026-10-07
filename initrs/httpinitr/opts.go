package httpinitr

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// resolvedConfig is private construction state owned by httpinitr.
type resolvedConfig struct {
	host              string
	port              int
	handler           http.Handler
	readHeaderTimeout *time.Duration
	readTimeout       *time.Duration
	writeTimeout      *time.Duration
	idleTimeout       *time.Duration
	maxHeaderBytes    int
	baseContext       func(net.Listener) context.Context
	serverOptions     []func(*http.Server) error
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
func WithConfig(config *ServerConfig) Option {
	return optionFunc(func(store *resolvedConfig) error {
		if config == nil {
			return nil
		}
		opts := []Option{
			WithHost(config.Host),
			WithPort(config.Port),
			WithMaxHeaderBytes(config.MaxHeaderBytes),
		}
		if config.ReadHeaderTimeout != nil {
			opts = append(opts, WithReadHeaderTimeout(seconds(*config.ReadHeaderTimeout)))
		}
		if config.ReadTimeout != nil {
			opts = append(opts, WithReadTimeout(seconds(*config.ReadTimeout)))
		}
		if config.WriteTimeout != nil {
			opts = append(opts, WithWriteTimeout(seconds(*config.WriteTimeout)))
		}
		if config.IdleTimeout != nil {
			opts = append(opts, WithIdleTimeout(seconds(*config.IdleTimeout)))
		}
		return apply(store, opts)
	})
}

func seconds(value int) time.Duration {
	return time.Duration(value) * time.Second
}

// WithHost sets the listening host. Empty listens on all interfaces.
func WithHost(host string) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.host = host
		return nil
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

// WithReadHeaderTimeout sets the maximum time to read request headers. It
// defaults to 10s; an explicit zero disables the limit.
func WithReadHeaderTimeout(timeout time.Duration) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.readHeaderTimeout = &timeout
		return nil
	})
}

// WithReadTimeout sets the maximum time to read the entire request including
// the body. It defaults to zero (no timeout) so streaming requests keep working.
func WithReadTimeout(timeout time.Duration) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.readTimeout = &timeout
		return nil
	})
}

// WithWriteTimeout sets the maximum time to write the response. It defaults to
// zero (no timeout) so streaming responses keep working.
func WithWriteTimeout(timeout time.Duration) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.writeTimeout = &timeout
		return nil
	})
}

// WithIdleTimeout sets how long an idle keep-alive connection is kept open. It
// defaults to 120s; an explicit zero disables the limit.
func WithIdleTimeout(timeout time.Duration) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.idleTimeout = &timeout
		return nil
	})
}

// WithMaxHeaderBytes sets the maximum size of request headers. Zero selects
// http.DefaultMaxHeaderBytes.
func WithMaxHeaderBytes(bytes int) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.maxHeaderBytes = bytes
		return nil
	})
}

// WithBaseContext sets the base context for incoming requests, exposing the
// native http.Server.BaseContext hook.
func WithBaseContext(base func(net.Listener) context.Context) Option {
	return optionFunc(func(store *resolvedConfig) error {
		store.baseContext = base
		return nil
	})
}

// WithNativeServer applies a deliberate escape hatch for server settings not
// represented by ServerConfig or the named options. It runs after the shell
// builds its server and may return an error.
func WithNativeServer(configure func(*http.Server) error) Option {
	return optionFunc(func(store *resolvedConfig) error {
		if configure != nil {
			store.serverOptions = append(store.serverOptions, configure)
		}
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
