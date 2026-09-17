// Package main is a minimal apin service: a zaal config file, three initrs
// (logging, postgres, grpc), and an apin.App owning the lifecycle.
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
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/zapinitr"
	"github.com/47monad/zaal"
)

func main() {
	ctx := context.Background()

	// Parse the config file. The env file is optional.
	cfg, err := zaal.New("config.json", ".env")
	if err != nil {
		log.Fatal(err)
	}

	// Logger initr: cross-cutting, returns apin.LoggerShell.
	loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
	if err != nil {
		log.Fatal(err)
	}

	app := apin.NewApp(apin.WithLogger(loggerShell.Logger))

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
