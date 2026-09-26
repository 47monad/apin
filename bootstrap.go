package apin

import "github.com/go-logr/logr"

// New creates an app and applies the same options as NewApp.
func New(opts ...AppOption) (*App, error) {
	return NewApp(opts...), nil
}

// MustNew is New without an error return.
func MustNew(opts ...AppOption) *App {
	return NewApp(opts...)
}

// SetLogger installs a logger for lifecycle events. It overrides any logger
// set during construction.
func (app *App) SetLogger(logger logr.Logger) {
	app.logger = logger
}

// RegisterLogger installs a logger for lifecycle events. Track a logger shell
// separately when it owns resources that must be closed.
func (app *App) RegisterLogger(logger logr.Logger) {
	if logger.GetSink() == nil {
		return
	}
	app.logger = logger
}
