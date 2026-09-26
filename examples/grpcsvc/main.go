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
)

type serviceConfig struct {
	Name     string          `json:"name" yaml:"name"`
	Logging  zapinitr.Config `json:"logging" yaml:"logging"`
	Postgres *pginitr.Config `json:"postgres" yaml:"postgres"`
	GRPC     *grpcConfig     `json:"grpc" yaml:"grpc"`
}

type grpcConfig struct {
	Servers map[string]grpcServerConfig `json:"servers" yaml:"servers"`
}

type grpcServerConfig struct {
	Port     int              `json:"port" yaml:"port"`
	Features grpcinitr.Config `json:"features" yaml:"features"`
}

func main() {
	ctx := context.Background()

	var cfg serviceConfig
	if err := config.Load("config.json", "", &cfg); err != nil {
		log.Fatal(err)
	}
	app := apin.NewApp()

	// Logger initr returns its own shell; App owns its lifecycle and shutdown.
	loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
	if err != nil {
		log.Fatal(err)
	}
	app.RegisterLogger(loggerShell.Logger)
	app.Track(loggerShell)

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
		grpcinitr.WithConfig(&grpcCfg.Features),
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
