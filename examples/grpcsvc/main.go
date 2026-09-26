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
	"net/http"
	"sort"

	"github.com/47monad/apin"
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/httpinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/zapinitr"
)

type serviceConfig struct {
	Name     string            `json:"name" yaml:"name"`
	Logging  zapinitr.Config   `json:"logging" yaml:"logging"`
	Postgres *pginitr.Config   `json:"postgres" yaml:"postgres"`
	GRPC     grpcinitr.Config  `json:"grpc" yaml:"grpc"`
	HTTP     serviceHTTPConfig `json:"http" yaml:"http"`
}

type serviceHTTPConfig struct {
	Servers map[string]httpinitr.Config `json:"servers" yaml:"servers"`
}

const grpcPort = 50051

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

	srvShell, err := grpcinitr.New(ctx,
		grpcinitr.WithConfig(&cfg.GRPC),
	)
	if err != nil {
		log.Fatal(err)
	}
	app.Track(srvShell)

	httpServerNames := make([]string, 0, len(cfg.HTTP.Servers))
	for name := range cfg.HTTP.Servers {
		httpServerNames = append(httpServerNames, name)
	}
	sort.Strings(httpServerNames)
	httpRunnables := make([]apin.Runnable, 0, len(httpServerNames))
	for _, name := range httpServerNames {
		serverConfig := cfg.HTTP.Servers[name]
		httpMux := http.NewServeMux()
		httpMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		httpShell, err := httpinitr.New(ctx,
			httpinitr.WithConfig(&serverConfig),
			httpinitr.WithHandler(httpMux),
		)
		if err != nil {
			log.Fatal(err)
		}
		app.Track(httpShell)

		httpListener, err := httpShell.Listen()
		if err != nil {
			log.Fatal(err)
		}
		httpRunnables = append(httpRunnables, func(ctx context.Context) error {
			return httpShell.Serve(ctx, httpListener)
		})
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatal(err)
	}

	// Register real services here, e.g. pb.RegisterUserServiceServer(s, ...)
	// inside a grpcinitr.WithRunnable option; the health server is already
	// registered by the initr.

	// Run until SIGINT/SIGTERM or failure; shells close in reverse order.
	runnables := []apin.Runnable{
		func(ctx context.Context) error { return srvShell.Serve(ctx, lis) },
	}
	runnables = append(runnables, httpRunnables...)
	if err := app.Run(ctx, runnables...); err != nil {
		loggerShell.Logger.Error(err, "application failed")
	}
}
