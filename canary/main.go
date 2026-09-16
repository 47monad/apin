// Package main is the compatibility canary of apin. It references every
// initr and every zaal config section so that a config schema change or an
// initr API change fails this build instead of a downstream service.
package main

import (
	"github.com/47monad/apin/initrs/etcdinitr"
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/mongoinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/prominitr"
	"github.com/47monad/apin/initrs/rmqinitr"
	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/47monad/zaal"
)

//nolint:unused // intentionally referenced to keep the canary compiling
func main() {
	cfg := &zaal.Config{
		Logging:    zaal.LoggingConfig{Level: "info"},
		Postgres:   &zaal.PostgresConfig{},
		Mongodb:    &zaal.MongodbConfig{},
		Etcd:       &zaal.EtcdConfig{},
		RabbiMQ:    &zaal.RabbitMQConfig{},
		Prometheus: &zaal.PrometheusConfig{},
		GRPC: &zaal.GRPCConfig{
			Servers: map[string]zaal.GRPCServerConfig{
				"default": {Features: zaal.GRPCFeatures{Reflection: true}},
			},
		},
	}
	grpcServer := cfg.GRPC.Servers["default"]

	_ = zapinitr.WithConfig(&cfg.Logging)
	_ = pginitr.WithConfig(cfg.Postgres)
	_ = mongoinitr.WithConfig(cfg.Mongodb)
	_ = etcdinitr.WithConfig(cfg.Etcd)
	_ = rmqinitr.WithConfig(cfg.RabbiMQ)
	_ = prominitr.WithConfig(cfg.Prometheus)
	_ = grpcinitr.WithConfig(&grpcServer)
}
