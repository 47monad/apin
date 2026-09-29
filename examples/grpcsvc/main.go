// Package main is a minimal service: the application defines the aggregate
// configuration it needs, and apin.App owns the lifecycle.
//
// Run it with a local postgres matching config.json (or set APIN_CONFIG to a
// JSON/YAML config path); SIGINT/SIGTERM shut everything down in reverse
// initialization order.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/47monad/apin"
	"github.com/47monad/apin/config"
	"github.com/47monad/apin/initrs/grpcinitr"
	"github.com/47monad/apin/initrs/httpinitr"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/apin/initrs/zapinitr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type serviceConfig struct {
	Name     string           `json:"name" yaml:"name"`
	Logging  zapinitr.Config  `json:"logging" yaml:"logging"`
	Postgres *pginitr.Config  `json:"postgres" yaml:"postgres"`
	GRPC     grpcinitr.Config `json:"grpc" yaml:"grpc"`
	HTTP     httpinitr.Config `json:"http" yaml:"http"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run returns any startup or run failure instead of exiting, so the single
// deferred App.Close below still releases every tracked shell when startup
// fails partway through.
func run() (err error) {
	ctx := context.Background()

	configPath := os.Getenv("APIN_CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}
	cfg, err := loadServiceConfig(configPath)
	if err != nil {
		return err
	}
	// Logger initr returns its own shell; App owns its lifecycle and shutdown.
	loggerShell, err := zapinitr.New(ctx, zapinitr.WithConfig(&cfg.Logging))
	if err != nil {
		return err
	}
	app := apin.New(apin.WithLogger(loggerShell.Logger))
	app.Track(loggerShell)

	// One bounded cleanup owns every shell tracked from here on, including the
	// shells registered before a startup failure. App.Run closes them itself on
	// the normal path; this Close is then a no-op.
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), app.ShutdownTimeout())
		defer cancel()
		if closeErr := app.Close(closeCtx); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	// Postgres: config-file values, with a per-field programmatic override.
	dbShell, err := pginitr.New(ctx,
		pginitr.WithConfig(cfg.Postgres),
		pginitr.WithSSLMode("disable"),
	)
	if err != nil {
		return err
	}
	app.Track(dbShell)
	readyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	pingErr := dbShell.Ping(readyCtx)
	cancel()
	if pingErr != nil {
		return fmt.Errorf("postgres is not ready: %w", pingErr)
	}

	// Query without branching on the shell mode.
	querier, err := dbShell.DB()
	if err != nil {
		return err
	}
	loggerShell.Logger.Info("postgres ready", "querier", fmt.Sprintf("%T", querier))

	grpcServerNames := make([]string, 0, len(cfg.GRPC.Servers))
	for name := range cfg.GRPC.Servers {
		grpcServerNames = append(grpcServerNames, name)
	}
	sort.Strings(grpcServerNames)
	grpcRunnables := make([]apin.Runnable, 0, len(grpcServerNames))
	for _, name := range grpcServerNames {
		serverConfig := cfg.GRPC.Servers[name]
		srvShell, err := grpcinitr.NewServer(ctx, serverConfig)
		if err != nil {
			return err
		}
		app.Track(srvShell)
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", srvShell.Port))
		if err != nil {
			return err
		}
		grpcRunnables = append(grpcRunnables, func(ctx context.Context) error {
			return srvShell.Serve(ctx, lis)
		})
	}

	grpcClientNames := make([]string, 0, len(cfg.GRPC.Clients))
	for name := range cfg.GRPC.Clients {
		grpcClientNames = append(grpcClientNames, name)
	}
	sort.Strings(grpcClientNames)
	for _, name := range grpcClientNames {
		clientShell, err := grpcinitr.NewClient(ctx, cfg.GRPC.Clients[name],
			// The sample uses plaintext credentials; real deployments should
			// supply their own native transport credentials.
			grpcinitr.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		)
		if err != nil {
			return err
		}
		app.Track(clientShell)
	}

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
		httpShell, err := httpinitr.NewServer(ctx, serverConfig,
			httpinitr.WithHandler(httpMux),
		)
		if err != nil {
			return err
		}
		app.Track(httpShell)

		httpListener, err := httpShell.Listen()
		if err != nil {
			return err
		}
		httpRunnables = append(httpRunnables, func(ctx context.Context) error {
			return httpShell.Serve(ctx, httpListener)
		})
	}

	// Run until SIGINT/SIGTERM or failure; shells close in reverse order.
	runnables := grpcRunnables
	runnables = append(runnables, httpRunnables...)
	if runErr := app.Run(ctx, runnables...); runErr != nil {
		loggerShell.Logger.Error(runErr, "application failed")
		return runErr
	}
	return nil
}

func loadServiceConfig(path string) (serviceConfig, error) {
	var cfg serviceConfig
	if err := config.Load(path, "", &cfg); err != nil {
		return serviceConfig{}, err
	}
	return cfg, nil
}
