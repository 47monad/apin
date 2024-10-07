package apin

import (
	"context"
	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"github.com/47monad/apin/internal/logger"
	"github.com/47monad/sercon"
)

type App struct {
	LoggerShell     *initr.LoggerShell
	Config          *sercon.Config
	MongodbShell    *initr.MongodbShell
	PrometheusShell *initr.PrometheusShell
	GrpcServerShell *initr.GrpcServerShell
}

func (app *App) GetName() string {
	return app.Config.Name
}

func (app *App) Logger() logger.Logger {
	return app.LoggerShell.Logger
}

func FromConfig(config *sercon.Config) *App {
	app := &App{
		Config: config,
	}

	return app
}

func (app *App) InitMongodb(ctx context.Context) error {
	b := initropts.Mongodb().SetUri(*app.Config.Mongodb.Uri)
	shell, err := initr.Mongodb(ctx, b)
	if err != nil {
		return err
	}
	shell.Db = shell.Client.Database(app.Config.Mongodb.DbName)
	app.MongodbShell = shell
	return nil
}

func (app *App) MustInitMongodb(ctx context.Context) {
	app.MustInit(ctx, app.InitMongodb)
}

func (app *App) InitPrometheus(ctx context.Context) error {
	b := initropts.Prometheus()
	shell, err := initr.Prometheus(ctx, b)
	if err != nil {
		return err
	}
	app.PrometheusShell = shell
	return nil
}

func (app *App) MustInitPrometheus(ctx context.Context) {
	app.MustInit(ctx, app.InitPrometheus)
}

func (app *App) MustInit(ctx context.Context, f func(context.Context) error) {
	err := f(ctx)
	if err != nil {
		panic(err)
	}
}

func (app *App) InitGrpc(ctx context.Context, opts *initropts.GrpcServerBuilder) error {
	if opts == nil {
		opts = initropts.GrpcServer()
	}
	if app.Config.Grpc.UseLogging {
		opts.SetLogging(app.Logger())
	}
	if app.Config.Grpc.UseReflection {
		opts.WithReflection()
	}
	if app.Config.Grpc.UseHealthCheck {
		opts.WithHealthCheck()
	}
	if app.Config.Prometheus.Enabled {
		opts.SetPrometheus(app.PrometheusShell.Registry)
	}

	shell, err := initr.GrpcServer(ctx, opts)
	if err != nil {
		return err
	}
	app.GrpcServerShell = shell
	return nil
}

func (app *App) MustInitGrpc(ctx context.Context, opts *initropts.GrpcServerBuilder) {
	err := app.InitGrpc(ctx, opts)
	if err != nil {
		panic(err)
	}
}

func (app *App) InitZap(ctx context.Context) error {
	shell, err := initr.Zap(ctx, initropts.Zap())
	if err != nil {
		return err
	}
	app.LoggerShell = shell
	return nil
}

func (app *App) MustInitZap(ctx context.Context) {
	app.MustInit(ctx, app.InitZap)
}
