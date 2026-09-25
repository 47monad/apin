package runner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/47monad/apin/manifest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// healthCheckTimeout bounds a single health check call, and makes it give up
// as soon as the runner starts shutting down.
const healthCheckTimeout = 2 * time.Second

// AddGRPCServer registers a gRPC server on the configured port. Stop drains
// its in-flight RPCs with GracefulStop.
func (r *Runner) AddGRPCServer(serverConfig *manifest.GRPCServerConfig, srv *grpc.Server) *Runner {
	if serverConfig == nil || srv == nil {
		r.logger.Error(nil, "AddGRPCServer: nil server config or server, skipping")
		return r
	}
	port := serverConfig.Port
	r.trackGRPCServer(srv)

	r.Add(func() error {
		r.logger.Info("starting grpc server", "port", port)
		err := serveOnPort(srv, port)
		if errors.Is(err, grpc.ErrServerStopped) {
			// GracefulStop or Stop closed the server; that is a clean exit.
			return nil
		}
		return err
	})
	return r
}

// AddHealthCheck polls checker and reflects the result in the gRPC health
// server until the runner context is done.
func (r *Runner) AddHealthCheck(hc *health.Server, interval time.Duration, checker func(context.Context) bool) *Runner {
	if hc == nil || checker == nil {
		r.logger.Error(nil, "AddHealthCheck: nil health server or checker, skipping")
		return r
	}

	r.Add(func() error {
		return runHealthChecker(r.ctx, r.name, hc, interval, func(hc *health.Server, setServing func(bool)) {
			ctx, cancel := context.WithTimeout(r.ctx, healthCheckTimeout)
			defer cancel()

			setServing(checker(ctx))
		})
	})
	return r
}

func serveOnPort(srv *grpc.Server, port int) error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	return srv.Serve(lis)
}

// gracefulStopGRPC drains in-flight RPCs, falling back to a hard Stop if ctx
// expires first. GracefulStop blocks until every RPC finishes, so it cannot
// simply be called with a context.
func gracefulStopGRPC(ctx context.Context, srv *grpc.Server) error {
	done := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// Drop whatever is still in flight so the drain cannot hang past
		// the caller's deadline. This unblocks GracefulStop.
		srv.Stop()
		<-done
		return fmt.Errorf("grpc server did not drain in time: %w", ctx.Err())
	}
}

// runHealthChecker polls on every tick until ctx is done. It runs the checks
// on the caller's goroutine, so a slow checker only delays shutdown.
func runHealthChecker(ctx context.Context, name string, hc *health.Server, interval time.Duration, cb func(*health.Server, func(bool))) error {
	_serving := true

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			cb(hc, func(serving bool) {
				if serving == _serving {
					return
				}
				if serving {
					hc.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
				} else {
					hc.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
				}
				_serving = serving
			})
		}
	}
}
