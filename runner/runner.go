// Package runner starts several servers and background tasks concurrently on
// a single errgroup, and shuts them down gracefully.
//
// Servers registered with [Runner.AddHTTPServer] and [Runner.AddGRPCServer]
// are tracked, so [Runner.Stop] drains in-flight requests and RPCs instead of
// dropping them. Stop also cancels the group context exposed by
// [Runner.Context], which is what unwinds the remaining runnables.
//
//	r := runner.New(ctx, "api", logger).
//		AddHTTPServer(cfg.HTTP.Servers["main"], attachRoutes).
//		AddGRPCServer(cfg.GRPC.Servers["main"], grpcSrv).
//		AddHealthCheck(healthSrv, time.Second, isHealthy)
//
//	if err := r.Run(); err != nil {
//		logger.Error(err, "runner stopped")
//	}
//
// # Shutdown on SIGINT/SIGTERM
//
// The Runner has no signal handling of its own; hand it the context that
// carries the signal and the servers are drained for you:
//
//	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
//	defer stop()
//
//	r := runner.New(sigCtx, "api", logger).AddHTTPServer(...)
//	go func() {
//		<-sigCtx.Done()
//		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//		defer cancel()
//		if err := r.Stop(shutdownCtx); err != nil {
//			logger.Error(err, "graceful shutdown failed")
//		}
//	}()
//
//	_ = r.Run() // returns once every runnable and server has finished
//
// Cancelling sigCtx alone is enough: Run drains the servers before it
// returns. Calling Stop is what bounds that drain with a deadline, so prefer
// it when the process has a shutdown budget.
//
// When the Runner is driven by an [github.com/47monad/apin.App], pass the
// App's context to New: App already handles signals and shell teardown, and
// the Runner will drain its servers as the App unwinds.
package runner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const defaultShutdownTimeout = 30 * time.Second

// Runner runs servers and background tasks concurrently. The zero value is
// not usable; construct one with [New].
type Runner struct {
	name   string
	logger logr.Logger
	eg     *errgroup.Group
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	timeout  time.Duration
	httpSrvs []*http.Server
	grpcSrvs []*grpc.Server

	stopOnce sync.Once
	stopErr  error
}

func New(ctx context.Context, name string, logger logr.Logger) *Runner {
	baseCtx, cancel := context.WithCancel(ctx)
	g, gctx := errgroup.WithContext(baseCtx)
	return &Runner{
		name:    name,
		logger:  logger,
		eg:      g,
		ctx:     gctx,
		cancel:  cancel,
		timeout: defaultShutdownTimeout,
	}
}

// Context is the errgroup context shared by the runner. It is cancelled when
// Stop is called, when a runnable fails, or when the context passed to New is
// cancelled. Long-running runnables should watch it.
func (r *Runner) Context() context.Context {
	return r.ctx
}

// SetShutdownTimeout bounds the drain that Stop performs when no deadline is
// available to it, i.e. when cancellation of the runner context is what
// triggered the shutdown. Defaults to 30s.
func (r *Runner) SetShutdownTimeout(timeout time.Duration) *Runner {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.timeout = timeout
	return r
}

// SetLimit bounds how many runnables execute at once. A negative value means
// no limit.
//
// Zero is rejected and ignored: errgroup reads it as "never start anything",
// so every Add after it would block forever. Use a negative value to run
// everything concurrently.
func (r *Runner) SetLimit(limit int) *Runner {
	if limit == 0 {
		r.logger.Error(nil, "SetLimit: ignoring limit of 0, it would block every runnable; pass a negative value for no limit",
			"limit", limit)
		return r
	}
	r.eg.SetLimit(limit)
	return r
}

// Add starts runnable as part of the group. A panic inside runnable is
// recovered and returned as an error, so one bad runnable reports through Run
// instead of taking the process down.
func (r *Runner) Add(runnable func() error) *Runner {
	if runnable == nil {
		return r
	}
	r.eg.Go(func() (err error) {
		defer func() {
			if p := recover(); p != nil {
				r.logger.Error(nil, "runnable panicked", "panic", p, "stack", string(debug.Stack()))
				err = fmt.Errorf("runner: runnable panicked: %v", p)
			}
		}()
		return runnable()
	})
	return r
}

// Run blocks until every runnable has returned, a runnable fails, or the
// context is cancelled, and then returns the first runnable error.
//
// Registered servers are drained as soon as the context is done, so a failing
// runnable cannot leave listeners open, and Run only returns once that drain
// has finished. A clean shutdown (Stop, parent context cancellation) returns
// nil.
func (r *Runner) Run() error {
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		<-r.ctx.Done()
		r.logger.Info("runner context done, draining servers")
		ctx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout())
		defer cancel()
		if err := r.Stop(ctx); err != nil {
			r.logger.Error(err, "runner drain failed")
		}
	}()

	err := r.eg.Wait()
	// eg.Wait cancels the group context, so the drain above is always
	// running; wait for it so Run never returns with servers still serving.
	<-drained
	return err
}

// Stop gracefully shuts down every registered server and then cancels the
// group context so the remaining runnables unwind. HTTP servers finish their
// in-flight requests and gRPC servers their in-flight RPCs; a gRPC server
// still busy when ctx expires is hard-stopped.
//
// Stop is idempotent and safe for concurrent use: later calls do not shut
// anything down twice and return the first call's error.
func (r *Runner) Stop(ctx context.Context) error {
	r.stopOnce.Do(func() {
		// Cancel even if the drain fails, otherwise runnables would keep
		// running with nothing left to serve.
		defer r.cancel()
		if ctx == nil {
			ctx = context.Background()
		}
		r.stopErr = r.shutdown(ctx)
	})
	return r.stopErr
}

func (r *Runner) shutdown(ctx context.Context) error {
	r.mu.Lock()
	httpSrvs := slices.Clone(r.httpSrvs)
	grpcSrvs := slices.Clone(r.grpcSrvs)
	r.httpSrvs, r.grpcSrvs = nil, nil
	r.mu.Unlock()

	if len(httpSrvs) == 0 && len(grpcSrvs) == 0 {
		return nil
	}
	r.logger.Info("shutting down runner", "httpServers", len(httpSrvs), "grpcServers", len(grpcSrvs))

	var errs []error
	for _, srv := range httpSrvs {
		// Shutdown returns the context error if the drain outlives ctx.
		if err := srv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http server shutdown: %w", err))
		}
	}
	for _, srv := range grpcSrvs {
		if err := gracefulStopGRPC(ctx, srv); err != nil {
			errs = append(errs, fmt.Errorf("grpc server shutdown: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (r *Runner) shutdownTimeout() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.timeout
}

func (r *Runner) trackHTTPServer(srv *http.Server) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.httpSrvs = append(r.httpSrvs, srv)
}

func (r *Runner) trackGRPCServer(srv *grpc.Server) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.grpcSrvs = append(r.grpcSrvs, srv)
}
