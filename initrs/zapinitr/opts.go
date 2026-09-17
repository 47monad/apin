package zapinitr

import (
	"github.com/47monad/apin/manifest"
)

// Store is the resolved configuration of a logger shell.
type Store struct {
	Level string
}

// Option mutates the store. Options are applied in the order they are passed
// to New, so later options win.
type Option func(*Store) error

// WithConfig applies a manifest config section. It is the entry point for
// config-file driven setups.
func WithConfig(config *manifest.LoggingConfig) Option {
	return func(s *Store) error {
		if config == nil {
			return nil
		}
		return apply(s, []Option{WithLevel(config.Level)})
	}
}

// WithLevel sets the log level (e.g. "debug", "info", "error"). Defaults to
// zap's production default.
func WithLevel(level string) Option {
	return func(s *Store) error {
		s.Level = level
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
