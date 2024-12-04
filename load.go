package apin

import (
	"errors"
	"fmt"
	"os"

	"github.com/47monad/sercon"
)

type LoadOptions struct {
	AppDir     string
	ConfigPath string
	Env        string
}

type LoadOption func(*LoadOptions)

func WithAppDir(name string) LoadOption {
	return func(lo *LoadOptions) {
		lo.AppDir = name
	}
}

func WithEnv(env string) LoadOption {
	return func(lo *LoadOptions) {
		lo.Env = env
	}
}

func Load(opts ...LoadOption) (*App, error) {
	_opts := &LoadOptions{
		ConfigPath: "./config/",
	}
	for _, o := range opts {
		o(_opts)
	}

	path := ".sercon/apin.json"
	if _opts.AppDir != "" {
		path = ".sercon/" + _opts.AppDir + "/apin.json"
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		configBasePath := _opts.ConfigPath + _opts.AppDir
		pklPath := fmt.Sprintf("%s/%s", configBasePath, "app.pkl")
		envPath := fmt.Sprintf("%s/%s.env", configBasePath, _opts.Env)
		if err := sercon.Build(pklPath, path, envPath); err != nil {
			return nil, err
		}
	}

	conf, err := getConfigFromJson(path)
	if err != nil {
		return nil, err
	}

	app := FromConfig(conf)

	return app, nil
}

func getConfigFromJson(path string) (*sercon.Config, error) {
	return sercon.LoadFromRelay(
		sercon.WithConfigPath(path),
	)
}
