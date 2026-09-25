// Package main is a minimal apin service: apin.New loads the service
// manifest, three initrs (logging, postgres, grpc) turn its config sections
// into shells, and the apin.App it returned owns the lifecycle.
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
)

func main() {
	ctx := context.Background()

	// Parse the config file. The env file is optional.
	app, err := apin.New(
		apin.WithConfig("config.json"),
		apin.WithEnv(".env"),
	)
	if err != nil {
		log.Fatal(err)
	}
	cfg := app.Config()

	// Logger initr: cross-cutting, returns apin.LoggerShell. It needs the
	// config apin.New just loaded, and the app needs its logger, so the two
	// meet here: RegisterLogger installs it after construction.
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
