package apin

import (
	"fmt"

	"github.com/47monad/apin/manifest"
	"github.com/go-logr/logr"
)

// Option configures New. Options are applied in the order they are passed,
// and the first error aborts New.
type Option func(*App) error

// WithConfig points New at the service manifest file (CUE). The manifest is
// validated against the built-in schema and stored on the app; read it back
// with App.Config and feed its sections to initrs via WithConfig.
func WithConfig(configPath string) Option {
	return func(a *App) error {
		a.configPath = configPath
		return nil
	}
}

// WithEnv points New at an optional .env file. It is loaded before the
// manifest is parsed, so its variables can override manifest values. Without
// WithEnv no env file is loaded.
func WithEnv(envPath string) Option {
	return func(a *App) error {
		a.envPath = envPath
		return nil
	}
}

// New initializes an app. Options such as WithConfig and WithEnv are applied
// in order; when a config path was given, the manifest is loaded after all
// options ran, so WithEnv takes effect regardless of option order.
//
// The app logger is typically not known at construction time — logger initrs
// need the loaded config — so register it afterwards with RegisterLogger.
func New(opts ...Option) (*App, error) {
	app := &App{
		logger:          logr.Discard(),
		shutdownTimeout: defaultShutdownTimeout,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(app); err != nil {
			return nil, err
		}
	}
	if app.configPath != "" {
		config, err := manifest.New(app.configPath, app.envPath)
		if err != nil {
			return nil, fmt.Errorf("apin: load manifest: %w", err)
		}
		app.config = config
	}
	return app, nil
}

// MustNew is New but panics on failure.
func MustNew(opts ...Option) *App {
	app, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return app
}

// Config returns the loaded manifest, or nil when New was called without
// WithConfig.
func (app *App) Config() *manifest.Config {
	return app.config
}

// SetLogger installs a logger for lifecycle events. It overrides any logger
// set during construction.
func (app *App) SetLogger(logger logr.Logger) {
	app.logger = logger
}

// RegisterLogger installs the logger of a logger initr shell (e.g. a
// zapinitr Shell) as the app logger. It is the post-construction counterpart
// of the WithLogger NewApp option.
func (app *App) RegisterLogger(shell *LoggerShell) {
	if shell == nil {
		return
	}
	app.logger = shell.Logger
}
