package zapinitr

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config contains zap initializer settings.
type Config struct {
	Level string `json:"level" yaml:"level" env:"log_level"`
}

// resolvedConfig is private construction state owned by zapinitr.
type resolvedConfig struct {
	zapConfig *zap.Config
	levelErr  error
}

// Option is a sealed functional option accepted by New.
type Option interface {
	apply(*resolvedConfig) error
}

type optionFunc func(*resolvedConfig) error

func (option optionFunc) apply(config *resolvedConfig) error {
	return option(config)
}

// WithConfig applies initializer-owned configuration.
func WithConfig(config *Config) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if config == nil {
			return nil
		}
		return apply(s, []Option{WithLevel(config.Level)})
	})
}

// WithLevel sets the log level (e.g. "debug", "info", "error"). Defaults to
// zap's production default.
func WithLevel(level string) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if level == "" {
			s.zapConfig.Level = zap.NewProductionConfig().Level
			s.levelErr = nil
			return nil
		}
		parsed, err := zapcore.ParseLevel(level)
		if err != nil {
			s.levelErr = fmt.Errorf("invalid zapinitr log level %q: %w", level, err)
			return nil
		}
		s.zapConfig.Level = zap.NewAtomicLevelAt(parsed)
		s.levelErr = nil
		return nil
	})
}

// WithNativeConfig customizes zap's native config for output, encoding,
// sampling, and other features not modeled by zapinitr.Config.
func WithNativeConfig(configure func(*zap.Config) error) Option {
	return optionFunc(func(s *resolvedConfig) error {
		if configure == nil {
			return nil
		}
		previousLevel := s.zapConfig.Level.Level()
		if err := configure(s.zapConfig); err != nil {
			return err
		}
		if s.zapConfig.Level.Level() != previousLevel {
			s.levelErr = nil
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
			errs = append(errs, fmt.Errorf("zapinitr: apply option: %w", err))
		}
	}
	return errors.Join(errs...)
}
