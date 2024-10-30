package apin

import (
	"fmt"

	"github.com/47monad/sercon"
)

type BootstrapOptions struct {
	BasePath       string
	ConfigPath     string
	EnvPath        string
	configFullPath string
	envFullPath    string
	UseRelay       bool
}

type BootstrapOption func(*BootstrapOptions)

func Bootstrap(options ...BootstrapOption) (*App, error) {
	opts := prepareOptions(options)
	conf, err := getConfig(opts)
	if err != nil {
		return nil, err
	}

	app := FromConfig(conf)

	return app, nil
}

func getConfig(opts *BootstrapOptions) (*sercon.Config, error) {
	if opts.UseRelay {
		return sercon.LoadFromRelay(
			sercon.WithConfigPath(opts.configFullPath),
		)
	}
	return sercon.Load(
		sercon.WithConfigPath(opts.configFullPath),
		sercon.WithEnvPath(opts.envFullPath),
	)
}

func prepareOptions(opts []BootstrapOption) *BootstrapOptions {
	basePath := "./config"
	newOpts := &BootstrapOptions{
		BasePath:   basePath,
		ConfigPath: "app.pkl",
		EnvPath:    ".env",
		UseRelay:   false,
	}

	for _, opt := range opts {
		opt(newOpts)
	}

	newOpts.configFullPath = fmt.Sprintf("%s/%s", newOpts.BasePath, newOpts.ConfigPath)
	newOpts.envFullPath = fmt.Sprintf("%s/%s", newOpts.BasePath, newOpts.EnvPath)

	return newOpts
}

func WithBasePath(path string) BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.BasePath = path
	}
}

func WithConfigPath(path string) BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.ConfigPath = path
	}
}

func WithEnvPath(path string) BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.EnvPath = path
	}
}

func UseRelay() BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.UseRelay = true
	}
}
