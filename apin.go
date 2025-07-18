package apin

import (
	"context"

	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"github.com/47monad/zaal"
	"github.com/go-logr/logr"
)

type App struct {
	LoggerShell *initr.LoggerShell
	config      *zaal.Config
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

func Must(err error) {
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
