package apin

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/47monad/apin/manifest"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

const defaultShutdownTimeout = 30 * time.Second

// App composes shells and owns their shutdown. It records every tracked
// shell and releases them in reverse tracking order, and blocks until a
// runnable fails, the context is cancelled, or SIGINT/SIGTERM arrives.
type App struct {
	logger          logr.Logger
	shutdownTimeout time.Duration
	config          *manifest.Config

	// configPath and envPath are recorded by the WithConfig/WithEnv options
	// and consumed once after all options are applied.
	configPath string
	envPath    string

	mu      sync.Mutex
	closers []Closer
}

// AppOption configures an App.
type AppOption func(*App)

// WithLogger sets the logger for lifecycle events. Defaults to discarding.
func WithLogger(logger logr.Logger) AppOption {
	return func(a *App) {
		a.logger = logger
	}
}

// WithShutdownTimeout bounds the shutdown phase. Defaults to 30s.
func WithShutdownTimeout(timeout time.Duration) AppOption {
	return func(a *App) {
		a.shutdownTimeout = timeout
	}
}

func NewApp(opts ...AppOption) *App {
	app := &App{
		logger:          logr.Discard(),
		shutdownTimeout: defaultShutdownTimeout,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(app)
	}
	return app
}

// Track records shells so Close releases them in reverse tracking order.
func (app *App) Track(shells ...Closer) *App {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, shell := range shells {
		if shell == nil {
			continue
		}
		app.closers = append(app.closers, shell)
	}
	return app
}

// Logger returns the app logger, handy for runners that take a logr.Logger.
func (app *App) Logger() logr.Logger {
	return app.logger
}

// SetShutdownTimeout bounds the shutdown phase performed by Run. It is the
// post-construction counterpart of the WithShutdownTimeout NewApp option, for
// apps built with New. Non-positive values restore the 30s default.
func (app *App) SetShutdownTimeout(timeout time.Duration) *App {
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.shutdownTimeout = timeout
	return app
}

// ShutdownTimeout reports the bound Run puts on its shutdown phase.
func (app *App) ShutdownTimeout() time.Duration {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.shutdownTimeout
}

// Runnable is a long-running unit of work. It must return promptly once its
// context is done.
type Runnable func(ctx context.Context) error

// Run starts the runnables and blocks until one of them fails, the context
// is cancelled, or SIGINT/SIGTERM is received. Runnables receive a context
// that is cancelled on any of those events. It then closes all tracked
// shells in reverse tracking order and returns the run and shutdown errors
// joined. The shutdown phase is bounded by the configured shutdown timeout.
//
// A context.Canceled from the run phase is not reported: cancellation by the
// app (signal, parent context) is a normal shutdown, not a failure.
func (app *App) Run(ctx context.Context, runnables ...Runnable) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg, egCtx := errgroup.WithContext(runCtx)
	eg.Go(func() error {
		defer cancel()
		return app.waitSignal(egCtx)
	})
	for _, runnable := range runnables {
		if runnable == nil {
			continue
		}
		eg.Go(func() error {
			return runnable(egCtx)
		})
	}
	runErr := eg.Wait()
	if errors.Is(runErr, context.Canceled) {
		runErr = nil
	}

	// A second signal during shutdown exits immediately.
	stop := make(chan struct{})
	defer close(stop)
	app.watchForceExit(stop)

	closeCtx, cancel := context.WithTimeout(context.Background(), app.ShutdownTimeout())
	defer cancel()
	closeErr := app.Close(closeCtx)

	return errors.Join(runErr, closeErr)
}

// Close releases all tracked shells in reverse tracking order. It is safe to
// call multiple times; subsequent calls are no-ops.
func (app *App) Close(ctx context.Context) error {
	app.mu.Lock()
	shells := app.closers
	app.closers = nil
	app.mu.Unlock()

	if len(shells) == 0 {
		return nil
	}
	app.logger.Info("shutting down", "shells", len(shells))

	var errs []error
	for i := len(shells) - 1; i >= 0; i-- {
		if err := shells[i].Close(ctx); err != nil {
			app.logger.Error(err, "shell shutdown failed")
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (app *App) waitSignal(ctx context.Context) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		return nil
	case sig := <-sigCh:
		app.logger.Info("shutdown signal received", "signal", sig.String())
		return nil
	}
}

// watchForceExit exits immediately on a second signal, before the watchdog
// channel is closed.
func (app *App) watchForceExit(stop chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer signal.Stop(sigCh)
		select {
		case <-stop:
		case sig := <-sigCh:
			app.logger.Info("forced exit on second signal", "signal", sig.String())
			os.Exit(1)
		}
	}()
}
