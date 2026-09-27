package grpcinitr

import (
	"context"
	"errors"
	"time"

	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
)

const healthCheckTimeout = 2 * time.Second

// RunHealthCheck periodically updates the registered gRPC health service.
// The checker should honor its context so it can be interrupted on shutdown.
func (shell *ServerShell) RunHealthCheck(ctx context.Context, service string, interval time.Duration, checker func(context.Context) bool) error {
	if shell == nil || shell.HealthServer == nil {
		return errors.New("grpcinitr: health checking is not enabled")
	}
	if interval <= 0 {
		return errors.New("grpcinitr: health check interval must be positive")
	}
	if checker == nil {
		return errors.New("grpcinitr: health checker is nil")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	serving := true
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
			isServing := checker(checkCtx)
			cancel()
			if ctx.Err() != nil {
				return nil
			}
			if isServing == serving {
				continue
			}
			status := healthgrpc.HealthCheckResponse_NOT_SERVING
			if isServing {
				status = healthgrpc.HealthCheckResponse_SERVING
			}
			shell.HealthServer.SetServingStatus(service, status)
			serving = isServing
		}
	}
}
