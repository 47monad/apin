package apin

import (
	"context"

	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"github.com/47monad/apin/internal/logger"
	"github.com/47monad/zaal"
)

type App struct {
	LoggerShell     *initr.LoggerShell
	config          *zaal.Config
	MongodbShell    *initr.MongodbShell
	PrometheusShell *initr.PrometheusShell
	GrpcServerShell *initr.GrpcServerShell
	RabbitMQShell   *initr.RabbitMQShell
}

func (app *App) GetName() string {
	return app.config.Name
}

func (app *App) GetConfig() *zaal.Config {
	return app.config
}

func (app *App) Logger() logger.Logger {
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

func (app *App) InitPrometheus(ctx context.Context) error {
	b := initropts.Prometheus()
	shell, err := initr.Prometheus(ctx, b)
	if err != nil {
		return err
	}
	app.PrometheusShell = shell
	return nil
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func (app *App) InitGrpc(ctx context.Context, opts *initropts.GrpcServerBuilder) error {
	if opts == nil {
		opts = initropts.GrpcServer()
	}
	if app.config.GRPC.Features.Logging {
		opts.SetLogging(app.Logger())
	}
	if app.config.GRPC.Features.Reflection {
		opts.WithReflection()
	}
	if app.config.GRPC.Features.HealthCheck {
		opts.WithHealthCheck()
	}
	if app.config.Prometheus != nil {
		opts.SetPrometheus(app.PrometheusShell.Registry)
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

func (app *App) InitRabbitMQ(ctx context.Context) error {
	shell, err := initr.RabbitMQ(ctx, initropts.RabbitMQ())
	if err != nil {
		return err
	}
	app.RabbitMQShell = shell
	return nil
}
