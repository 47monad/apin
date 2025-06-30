package apin

import (
	"context"

	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"github.com/47monad/zaal"
	"github.com/go-logr/logr"
)

type App struct {
	LoggerShell     *initr.LoggerShell
	config          *zaal.Config
	MongodbShell    *initr.MongodbShell
	GrpcServerShell *initr.GrpcServerShell
}

func (app *App) GetName() string {
	return app.config.Name
}

func (app *App) GetConfig() *zaal.Config {
	return app.config
}

func (app *App) Logger() logr.Logger {
	return app.LoggerShell.Logger
}

func FromConfig(config *zaal.Config) *App {
	app := &App{
		config: config,
	}

	return app
}

func (app *App) InitMongodb(ctx context.Context) error {
	b := initropts.Mongodb().SetUri(app.config.Mongodb.URI)
	shell, err := initr.Mongodb(ctx, b)
	if err != nil {
		return err
	}
	shell.Db = shell.Client.Database(app.config.Mongodb.DbName)
	app.MongodbShell = shell
	return nil
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func (app *App) InitGrpc(ctx context.Context, name string, opts *initropts.GrpcServerBuilder) error {
	if opts == nil {
		opts = initropts.GrpcServer()
	}
	server := app.config.GRPC.Servers[name]
	if server.Features.Logging {
		opts.SetLogging(app.Logger())
	}
	if server.Features.Reflection {
		opts.WithReflection()
	}
	if server.Features.HealthCheck {
		opts.WithHealthCheck()
	}

	shell, err := initr.GrpcServer(ctx, opts)
	if err != nil {
		return err
	}
	app.GrpcServerShell = shell
	return nil
}

func (app *App) InitZap(ctx context.Context) error {
	shell, err := initr.Zap(ctx, initropts.Zap())
	if err != nil {
		return err
	}
	app.LoggerShell = shell
	return nil
}
