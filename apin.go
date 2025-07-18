package apin

import (
	"github.com/47monad/zaal"
)

type App struct {
	config *zaal.Config
}

func (app *App) GetName() string {
	return app.config.Name
}

func (app *App) GetConfig() *zaal.Config {
	return app.config
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
