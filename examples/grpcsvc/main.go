// Package main is a minimal service: the application defines the aggregate
// configuration it needs, and apin.App owns the lifecycle.
//
// Run it with a local postgres matching config.json; SIGINT/SIGTERM shut
// everything down in reverse initialization order.
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/47monad/apin"
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/47monad/apin/manifest"
)

type serviceConfig struct {
	Name     string                 `json:"name" yaml:"name"`
	Logging  manifest.LoggingConfig `json:"logging" yaml:"logging"`
	Postgres *pginitr.Config        `json:"postgres" yaml:"postgres"`
	GRPC     *manifest.GRPCConfig   `json:"grpc" yaml:"grpc"`
}

func main() {
	ctx := context.Background()

	var cfg serviceConfig
	if err := config.Load("config.json", "", &cfg); err != nil {
		log.Fatal(err)
	}
	app := apin.NewApp()

	// Logger initr: cross-cutting, returns apin.LoggerShell. RegisterLogger
	// installs it for lifecycle events.
	loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
	if err != nil {
		log.Fatal(err)
	}
	app.RegisterLogger(loggerShell)

	// Postgres: config-file values, with a per-field programmatic override.
	dbShell, err := pginitr.New(ctx,
		pginitr.WithConfig(cfg.Postgres),
		pginitr.WithSSLMode("disable"),
	)
	if err != nil {
		log.Fatal(err)
	}
	app.Track(dbShell)

	// Query without branching on the shell mode.
	querier, err := dbShell.DB()
	if err != nil {
		log.Fatal(err)
	}
	loggerShell.Logger.Info("postgres ready", "querier", fmt.Sprintf("%T", querier))

	grpcCfg := cfg.GRPC.Servers["api"]
	srvShell, err := grpcinitr.New(ctx,
		grpcinitr.WithConfig(&grpcCfg),
	)
	if err != nil {
		log.Fatal(err)
	}
	app.Track(srvShell)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcCfg.Port))
	if err != nil {
		log.Fatal(err)
	}

	// Register real services here, e.g. pb.RegisterUserServiceServer(s, ...)
	// inside a grpcinitr.WithRunnable option; the health server is already
	// registered by the initr.

	// Run until SIGINT/SIGTERM or failure; shells close in reverse order.
	if err := app.Run(ctx, func(ctx context.Context) error {
		return srvShell.Serve(ctx, lis)
	}); err != nil {
		loggerShell.Logger.Error(err, "application failed")
	}
}
