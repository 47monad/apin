package runner

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/47monad/apin/internal/logger"
	"github.com/47monad/zaal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func (r *Runner) AddGrpcServer(serverConfig *zaal.GRPCServerConfig, srv *grpc.Server) {
	r.eg.Go(func() error {
		port := serverConfig.Port
		r.app.Logger().Info("starting grpc server", logger.LogFields{"port": port})
		return serveOnPort(srv, port)
	})
}

func (r *Runner) AddHealthCheck(hc *health.Server, interval time.Duration, checker func(context.Context) bool) {
	r.eg.Go(func() error {
		runHealthChecker(r.app.GetName(), hc, interval, func(hc *health.Server, setServing func(bool)) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			isHealthy := checker(ctx)
			if isHealthy {
				setServing(false)
			} else {
				setServing(true)
			}
		})
		return nil
	})
}

func serveOnPort(srv *grpc.Server, port int) error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}
	if err := srv.Serve(lis); err != nil {
		return err
	}
	return nil
}

func runHealthChecker(name string, hc *health.Server, interval time.Duration, cb func(*health.Server, func(bool))) {
	_serving := true

	t := time.NewTicker(interval)
	defer t.Stop()

	go func() {
		for {
			select {
			case <-t.C:
				cb(hc, func(serving bool) {
					if serving != _serving {
						if serving {
							hc.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_SERVING)
						} else {
							hc.SetServingStatus(name, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
						}
						_serving = serving
					}
				})
			}
		}
	}()

	select {}
}
